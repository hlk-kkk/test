"""LLM-based topic discovery and classification."""

import asyncio
import json
import logging
import random
from dataclasses import dataclass

from .data_loader import Candidate
from .llm_client import chat

logger = logging.getLogger(__name__)


@dataclass
class Category:
    name: str  # snake_case, e.g. "fashion_portrait"
    display_name: str  # e.g. "Fashion & Portrait"
    description: str


DISCOVER_PROMPT = """Analyze these {count} AI image generation prompts collected from X/Twitter.
Identify 5-8 natural topic categories that best organize them.

Requirements:
- Categories should be distinct with minimal overlap
- Each category should have a reasonable number of prompts (not too niche)
- Cover the main themes you observe in the data

For each category provide:
- name: English, lowercase, snake_case (e.g. "fashion_portrait")
- display_name: English title case (e.g. "Fashion & Portrait")
- description: One sentence describing what belongs in this category

Output ONLY a JSON array:
[{{"name": "...", "display_name": "...", "description": "..."}}, ...]

Here are the prompts:

{prompts}"""


CLASSIFY_PROMPT = """Classify each prompt into exactly one of these categories:

{categories}

For each prompt, pick the single best-fitting category.
Output ONLY a JSON array:
[{{"tweet_id": "...", "category": "category_name"}}, ...]

Prompts:

{prompts}"""


async def discover_categories(
    candidates: list[Candidate],
    sample_size: int = 200,
) -> list[Category]:
    """Sample prompts and ask LLM to discover natural categories."""
    sample = random.sample(candidates, min(sample_size, len(candidates)))

    # Truncate long prompts to save tokens
    prompt_texts = []
    for c in sample:
        text = c.prompt_text[:500] if len(c.prompt_text) > 500 else c.prompt_text
        prompt_texts.append(f"[{c.tweet_id}] {text}")

    prompt = DISCOVER_PROMPT.format(
        count=len(prompt_texts),
        prompts="\n---\n".join(prompt_texts),
    )

    response = await chat(
        messages=[{"role": "user", "content": prompt}],
        temperature=0.3,
        max_tokens=2048,
    )

    # Parse JSON from response (handle markdown code blocks)
    text = response.strip()
    if text.startswith("```"):
        text = text.split("\n", 1)[1]
        text = text.rsplit("```", 1)[0]

    categories = [Category(**item) for item in json.loads(text)]
    logger.info("Discovered %d categories: %s", len(categories),
                [c.name for c in categories])
    return categories


async def classify_candidates(
    candidates: list[Candidate],
    categories: list[Category],
    batch_size: int = 50,
    concurrency: int = 5,
) -> dict[str, str]:
    """Classify all candidates into categories. Returns {tweet_id: category_name}."""
    cat_desc = "\n".join(
        f"- {c.name}: {c.description}" for c in categories
    )

    semaphore = asyncio.Semaphore(concurrency)
    results: dict[str, str] = {}

    async def classify_batch(batch: list[Candidate]) -> None:
        prompt_texts = []
        for c in batch:
            text = c.prompt_text[:300] if len(c.prompt_text) > 300 else c.prompt_text
            prompt_texts.append(f"[{c.tweet_id}] {text}")

        prompt = CLASSIFY_PROMPT.format(
            categories=cat_desc,
            prompts="\n---\n".join(prompt_texts),
        )

        async with semaphore:
            try:
                response = await chat(
                    messages=[{"role": "user", "content": prompt}],
                    temperature=0.1,
                    max_tokens=4096,
                )

                text = response.strip()
                if text.startswith("```"):
                    text = text.split("\n", 1)[1]
                    text = text.rsplit("```", 1)[0]

                classified = json.loads(text)
                valid_names = {c.name for c in categories}
                for item in classified:
                    cat = item["category"]
                    if cat in valid_names:
                        results[item["tweet_id"]] = cat
                    else:
                        results[item["tweet_id"]] = categories[-1].name  # fallback
            except Exception:
                logger.exception("Failed to classify batch of %d", len(batch))
                # Assign fallback category
                for c in batch:
                    results[c.tweet_id] = categories[-1].name

    # Process in batches
    batches = [
        candidates[i:i + batch_size]
        for i in range(0, len(candidates), batch_size)
    ]

    tasks = [classify_batch(batch) for batch in batches]
    await asyncio.gather(*tasks)

    logger.info("Classified %d/%d candidates", len(results), len(candidates))
    return results
