"""Load candidates from xprompt-watch SQLite database."""

import logging
import sqlite3
from dataclasses import dataclass, field
from pathlib import Path

logger = logging.getLogger(__name__)

XPROMPT_DB_DEFAULT = "/Users/nanmi/workspace/myself_code/xprompt-watch/db/prompts.db"
XPROMPT_IMAGES_DIR = "/Users/nanmi/workspace/myself_code/xprompt-watch/images"


@dataclass
class Candidate:
    tweet_id: str
    author: str
    tool: str
    prompt_text: str
    tweet_url: str
    image_paths: list[str] = field(default_factory=list)
    category: str = ""
    wins: int = 0
    losses: int = 0
    is_bye: bool = False  # marked True if won by bye round


def load_candidates(db_path: str = XPROMPT_DB_DEFAULT) -> list[Candidate]:
    """Load all prompts with at least one existing image file."""
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()

    cursor.execute("""
        SELECT p.tweet_id, p.author, p.tool, p.prompt_text, p.tweet_url,
               GROUP_CONCAT(i.local_path, '|||') as image_paths
        FROM prompts p
        JOIN images i ON p.tweet_id = i.tweet_id
        WHERE p.prompt_text IS NOT NULL AND p.prompt_text != ''
        GROUP BY p.tweet_id
        HAVING COUNT(i.id) > 0
        ORDER BY p.created_at DESC
    """)

    candidates = []
    images_dir = Path(XPROMPT_IMAGES_DIR)

    for row in cursor.fetchall():
        raw_paths = row["image_paths"].split("|||") if row["image_paths"] else []
        # Resolve paths and filter to existing files
        valid_paths = []
        for p in raw_paths:
            full_path = images_dir.parent / p if not Path(p).is_absolute() else Path(p)
            if full_path.exists():
                valid_paths.append(str(full_path))

        if not valid_paths:
            continue

        candidates.append(Candidate(
            tweet_id=row["tweet_id"],
            author=row["author"] or "unknown",
            tool=row["tool"] or "unknown",
            prompt_text=row["prompt_text"],
            tweet_url=row["tweet_url"] or "",
            image_paths=valid_paths,
        ))

    conn.close()
    logger.info("Loaded %d candidates with valid images from %s", len(candidates), db_path)
    return candidates
