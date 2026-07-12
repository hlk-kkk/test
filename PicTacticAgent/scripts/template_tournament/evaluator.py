"""Vision LLM 1v1 tournament bracket evaluation."""

import asyncio
import csv
import json
import logging
import random
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path

from .data_loader import Candidate
from .llm_client import chat_with_images

logger = logging.getLogger(__name__)

JUDGE_PROMPT = """You are judging an AI image generation tournament.
Compare Image A and Image B on these criteria:
1. Composition and layout
2. Aesthetic quality and color harmony
3. Detail and technical quality
4. Visual impact and creativity
5. Overall polish

Which image is better? Reply ONLY with JSON (no other text):
{{"winner": "A" or "B", "confidence": 0.5 to 1.0, "reason": "brief explanation"}}"""


@dataclass
class MatchResult:
    round_num: int
    category: str
    match_num: int
    winner_id: str
    loser_id: str
    confidence: float
    reason: str
    timestamp: str


class TournamentLog:
    """Append-only TSV log of match results."""

    def __init__(self, path: str | Path):
        self.path = Path(path)
        self._ensure_header()

    def _ensure_header(self) -> None:
        if not self.path.exists():
            with open(self.path, "w", newline="") as f:
                writer = csv.writer(f, delimiter="\t")
                writer.writerow([
                    "round", "category", "match", "winner_id",
                    "loser_id", "confidence", "reason", "timestamp",
                ])

    def log(self, result: MatchResult) -> None:
        with open(self.path, "a", newline="") as f:
            writer = csv.writer(f, delimiter="\t")
            writer.writerow([
                result.round_num, result.category, result.match_num,
                result.winner_id, result.loser_id,
                result.confidence, result.reason, result.timestamp,
            ])


async def judge_match(
    a: Candidate,
    b: Candidate,
    semaphore: asyncio.Semaphore,
) -> tuple[str, float, str]:
    """Run a 1v1 match with position swap for bias reduction.

    Returns (winner_tweet_id, confidence, reason).
    """
    img_a = a.image_paths[0]
    img_b = b.image_paths[0]

    async def _judge(first_path: str, second_path: str) -> dict:
        async with semaphore:
            try:
                response = await chat_with_images(
                    text_prompt=JUDGE_PROMPT,
                    image_paths=[first_path, second_path],
                    temperature=0.2,
                    max_tokens=512,
                )
                text = response.strip()
                if text.startswith("```"):
                    text = text.split("\n", 1)[1]
                    text = text.rsplit("```", 1)[0]
                return json.loads(text)
            except Exception:
                logger.exception("Judge call failed for %s vs %s", a.tweet_id, b.tweet_id)
                return {"winner": "A", "confidence": 0.5, "reason": "judge error, random pick"}

    # Run forward and reverse
    result_forward = await _judge(img_a, img_b)
    result_reverse = await _judge(img_b, img_a)

    # Map results to actual candidates
    fwd_winner = a.tweet_id if result_forward["winner"] == "A" else b.tweet_id
    rev_winner = b.tweet_id if result_reverse["winner"] == "A" else a.tweet_id

    if fwd_winner == rev_winner:
        # Consistent — confident result
        confidence = (result_forward.get("confidence", 0.7)
                      + result_reverse.get("confidence", 0.7)) / 2
        reason = result_forward.get("reason", "")
        return fwd_winner, confidence, reason
    else:
        # Inconsistent — tie, pick randomly
        winner = random.choice([a.tweet_id, b.tweet_id])
        return winner, 0.5, "tie (inconsistent judgments), random pick"


async def run_tournament(
    candidates: list[Candidate],
    category: str,
    target_count: int,
    log: TournamentLog,
    concurrency: int = 5,
) -> list[Candidate]:
    """Run elimination tournament within one category until target_count remain."""
    if len(candidates) <= target_count:
        logger.info("Category '%s': %d candidates <= target %d, all advance",
                     category, len(candidates), target_count)
        return candidates

    semaphore = asyncio.Semaphore(concurrency)
    current = list(candidates)
    random.shuffle(current)
    round_num = 0

    while len(current) > target_count:
        round_num += 1

        # If a full halving round would drop below target, sort by win rate instead
        next_size = (len(current) + 1) // 2  # ceiling division
        if next_size < target_count:
            logger.info("Category '%s' Round %d: %d candidates — selecting top %d by win rate",
                         category, round_num, len(current), target_count)
            current.sort(key=lambda c: (c.wins / max(1, c.wins + c.losses), c.wins),
                         reverse=True)
            return current[:target_count]

        logger.info("Category '%s' Round %d: %d candidates",
                     category, round_num, len(current))

        # Pair up
        pairs: list[tuple[Candidate, Candidate]] = []
        byes: list[Candidate] = []

        for i in range(0, len(current) - 1, 2):
            pairs.append((current[i], current[i + 1]))
        if len(current) % 2 == 1:
            bye = current[-1]
            bye.is_bye = True
            byes.append(bye)

        # Run all matches concurrently
        async def run_match(
            pair: tuple[Candidate, Candidate], match_num: int
        ) -> Candidate:
            a, b = pair
            winner_id, confidence, reason = await judge_match(a, b, semaphore)
            loser_id = b.tweet_id if winner_id == a.tweet_id else a.tweet_id

            log.log(MatchResult(
                round_num=round_num,
                category=category,
                match_num=match_num,
                winner_id=winner_id,
                loser_id=loser_id,
                confidence=confidence,
                reason=reason,
                timestamp=datetime.now(timezone.utc).isoformat(),
            ))

            winner = a if winner_id == a.tweet_id else b
            winner.wins += 1
            loser = b if winner_id == a.tweet_id else a
            loser.losses += 1
            return winner

        tasks = [
            run_match(pair, i + 1)
            for i, pair in enumerate(pairs)
        ]
        winners = await asyncio.gather(*tasks)
        current = list(winners) + byes

        logger.info("Category '%s' Round %d: %d → %d",
                     category, round_num, len(pairs) * 2 + len(byes), len(current))

    return current
