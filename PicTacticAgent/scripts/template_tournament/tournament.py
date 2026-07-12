"""Template Tournament — main entry point.

Usage:
    cd /Users/nanmi/workspace/myself_code/PicTacticAgent
    uv run python -m scripts.template_tournament.tournament [--target-count 50] [--concurrency 5]
"""

import argparse
import asyncio
import logging
import sys
from collections import defaultdict
from pathlib import Path

from .classifier import classify_candidates, discover_categories
from .data_loader import load_candidates
from .evaluator import TournamentLog, run_tournament
from .exporter import generate_metadata, generate_seed_module, migrate_images

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(name)s %(levelname)s %(message)s",
)
logger = logging.getLogger(__name__)

RESULTS_TSV = Path(__file__).parent / "results.tsv"


async def main(target_count: int = 50, concurrency: int = 5, db_path: str = "") -> None:
    # Stage 1: Load
    logger.info("=== Stage 1: Loading candidates ===")
    kwargs = {}
    if db_path:
        kwargs["db_path"] = db_path
    candidates = load_candidates(**kwargs)
    logger.info("Loaded %d candidates", len(candidates))

    if len(candidates) == 0:
        logger.error("No candidates found. Exiting.")
        sys.exit(1)

    # Stage 2: Discover categories
    logger.info("=== Stage 2: Discovering categories ===")
    categories = await discover_categories(candidates, sample_size=200)
    for cat in categories:
        logger.info("  Category: %s — %s", cat.name, cat.description)

    # Stage 3: Classify all candidates
    logger.info("=== Stage 3: Classifying candidates ===")
    classification = await classify_candidates(
        candidates, categories, batch_size=50, concurrency=concurrency,
    )

    # Assign categories to candidates
    for c in candidates:
        c.category = classification.get(c.tweet_id, categories[-1].name)

    # Group by category
    by_category: dict[str, list] = defaultdict(list)
    for c in candidates:
        by_category[c.category].append(c)

    logger.info("Distribution: %s",
                {k: len(v) for k, v in by_category.items()})

    # Stage 4: Tournament per category
    logger.info("=== Stage 4: Running tournaments ===")
    tournament_log = TournamentLog(RESULTS_TSV)

    # Allocate quotas proportionally, min 3 per category
    total_candidates = sum(len(v) for v in by_category.values())
    quotas: dict[str, int] = {}
    for cat_name, members in by_category.items():
        quota = max(3, round(target_count * len(members) / total_candidates))
        quotas[cat_name] = quota

    # Adjust to hit target_count
    total_quota = sum(quotas.values())
    if total_quota > target_count:
        # Trim from largest categories
        sorted_cats = sorted(quotas.keys(), key=lambda k: quotas[k], reverse=True)
        for cat_name in sorted_cats:
            if total_quota <= target_count:
                break
            reduction = min(quotas[cat_name] - 3, total_quota - target_count)
            quotas[cat_name] -= reduction
            total_quota -= reduction

    logger.info("Quotas: %s (total: %d)", quotas, sum(quotas.values()))

    all_winners: list = []
    for cat_name, members in by_category.items():
        quota = quotas.get(cat_name, 3)
        winners = await run_tournament(
            members, cat_name, quota, tournament_log, concurrency=concurrency,
        )
        all_winners.extend(winners)
        logger.info("Category '%s': %d winners from %d candidates",
                     cat_name, len(winners), len(members))

    logger.info("Total winners: %d", len(all_winners))

    # Stage 5: Generate metadata
    logger.info("=== Stage 5: Generating metadata ===")
    metadata = await generate_metadata(all_winners, concurrency=concurrency)

    # Stage 6: Export
    logger.info("=== Stage 6: Exporting ===")
    image_urls = migrate_images(all_winners)
    generate_seed_module(all_winners, metadata, image_urls, categories)

    logger.info("=== DONE ===")
    logger.info("Results log: %s", RESULTS_TSV)
    logger.info("Seed module: backend/src/pictactic/db/system_templates.py")
    logger.info("Images: storage/templates/")


def cli() -> None:
    parser = argparse.ArgumentParser(description="Template Tournament")
    parser.add_argument("--target-count", type=int, default=50,
                        help="Target number of templates (default: 50)")
    parser.add_argument("--concurrency", type=int, default=5,
                        help="Max concurrent LLM calls (default: 5)")
    parser.add_argument("--xprompt-db", type=str, default="",
                        help="Path to xprompt-watch prompts.db")
    args = parser.parse_args()
    asyncio.run(main(
        target_count=args.target_count,
        concurrency=args.concurrency,
        db_path=args.xprompt_db,
    ))


if __name__ == "__main__":
    cli()
