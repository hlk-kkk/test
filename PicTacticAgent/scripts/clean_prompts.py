"""One-shot script to clean Twitter/social media metadata from system template prompts."""

import re
import sys
from pathlib import Path

# Add project root to path
sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from backend.src.pictactic.db.system_templates import SYSTEM_TEMPLATES


def clean_prompt(text: str) -> str:
    """Remove social media metadata from prompt text, keeping only the actual prompt."""
    original = text
    text = text.strip()

    # --- Strategy 0: If prompt has JSON body with non-JSON intro, extract JSON ---
    json_start = text.find("{")
    if json_start > 0:
        before_json = text[:json_start].strip()
        json_part = text[json_start:]
        # If there's significant social media content before JSON, remove it
        if (
            json_part.rstrip().endswith("}")
            and len(json_part) > 100
            and any(
                kw in before_json.lower()
                for kw in [
                    "nano banana", "@", "prompt", "share", "grok",
                    "faces glowing", "karina", "aespa",
                ]
            )
        ):
            text = json_part

    # If it's pure JSON format, return as-is
    stripped = text.strip()
    if stripped.startswith("{") and stripped.endswith("}"):
        return stripped

    # --- Strategy 1: Find explicit "Prompt" markers and take everything after ---
    # Handles: "Prompt ⤵️\n\n...", "Prompt👇\n\n...", "Prompt\n...", "Prompt;\n..."
    prompt_marker = re.search(
        r"(?:^|\n)Prompt\s*[⤵️👇🔽↓;:]*\s*\n+", text, flags=re.UNICODE
    )
    if prompt_marker:
        after = text[prompt_marker.end() :]
        if len(after.strip()) > 50:
            text = after

    # Strip so subsequent ^ patterns work on actual start of content
    text = text.strip()

    # --- Strategy 2: Remove known social media prefix patterns ---

    # Lines with emoji-wrapped "Prompt Share" headers (works on any line start)
    text = re.sub(
        r"^[^\w\s]*Prompt\s*Share[^\n]*\n"
        r"(?:[^\n]*(?:Use this prompt|Experiment|share your|create something|create your)[^\n]*\n)*"
        r"\n*",
        "",
        text,
        flags=re.IGNORECASE | re.MULTILINE,
    )

    # "(Use this prompt...)" standalone lines
    text = re.sub(r"^\(?Use this prompt[^\n]*\)?\s*\n\n?", "", text, flags=re.MULTILINE)

    # Lines mentioning "Nano Banana" (tool references)
    text = re.sub(r"^[^\n]*[Nn]ano\s*[Bb]anana[^\n]*\n\n?", "", text, flags=re.MULTILINE)

    # "Prompt Studio: ..., with Nano Banana..." intro lines
    text = re.sub(r"^Prompt Studio:[^\n]*\n\n?", "", text, flags=re.MULTILINE)

    # "Google Nano Banana Pro on @..." intros
    text = re.sub(r"^Google\s+Nano[^\n]*\n\n?", "", text, flags=re.MULTILINE)

    # "Meta AI vs Nano Banana..." comparison intros
    text = re.sub(r"^Meta AI vs[^\n]*\n\n?", "", text, flags=re.MULTILINE)

    # "Difference between Grok..." comparison intros
    text = re.sub(r"^Difference between[^\n]*\n", "", text, flags=re.MULTILINE)

    # "Recommended generator:..." lines
    text = re.sub(r"^Recommended generator:[^\n]*\n\n?", "", text, flags=re.MULTILINE)

    # "Try it and show your results..." lines
    text = re.sub(r"^Try it and show[^\n]*\n\n?", "", text, flags=re.MULTILINE)

    # Lines that are just social context: "Chaitra Navratri with AI 🙏"
    text = re.sub(r"^[^\n]*with AI\s*[\U0001f300-\U0001f9ff]*\s*\n\n?", "", text, flags=re.MULTILINE)

    # "Gemini Nano Banana Pro 🍌" type lines
    text = re.sub(r"^[^\n]*Gemini\s+Nano[^\n]*\n\n?", "", text, flags=re.MULTILINE)

    # "The moment your Beast escapes... #AKCB" (social intro with hashtag)
    text = re.sub(r"^The moment your[^\n]*#\w+[^\n]*\n\n?", "", text, flags=re.MULTILINE)

    # Non-English social media lines (Turkish, etc.) before actual prompt
    text = re.sub(
        r"^[^\n]*(?:indirmeye gerek|Beğendiğini|logoyu sil)[^\n]*\n\n?",
        "",
        text,
        flags=re.MULTILINE,
    )

    # "@username" mention lines at start (when they're just attribution)
    text = re.sub(r"^@\w+\s*\n", "", text, flags=re.MULTILINE)

    # "Now i have a reason to ditch..." type tweet-only content
    text = re.sub(r"^Now [iI] have a reason[^\n]*$", "", text, flags=re.MULTILINE)

    # "Same prompt for Grok imagine and..." lines
    text = re.sub(r"^Same prompt for[^\n]*\n\n?", "", text, flags=re.MULTILINE)

    # "Exceptional Realism:..." commentary lines
    text = re.sub(r"^Exceptional Realism:[^\n]*\n\n?", "", text, flags=re.MULTILINE)

    # "Image by @..." attribution lines
    text = re.sub(r"^[^\n]*image by @\w+[^\n]*\n\n?", "", text, flags=re.IGNORECASE | re.MULTILINE)

    # "Original Prompt by @..." attribution lines
    text = re.sub(r"^Original Prompt by @\w+\s*\n\n?", "", text, flags=re.MULTILINE)

    # Lines that are entirely hashtags and/or URLs (no real content)
    text = re.sub(r"^(?:#\w+\s*|https?://\S+\s*|@\w+\s*)+$", "", text, flags=re.MULTILINE)

    # "Fibonacci version :" and similar sub-labels
    text = re.sub(r"^Fibonacci version\s*:\s*\n\n?", "", text, flags=re.MULTILINE)

    # Quoted tweet blocks: "> QT @username:\n> ..."
    text = re.sub(r"^>.*\n(?:>.*\n)*\n*", "", text, flags=re.MULTILINE)

    # "💬Prompt:" labels
    text = re.sub(r"^[^\w\s]*Prompt[;:]\s*\n\n?", "", text, flags=re.MULTILINE)

    # --- Strategy 3: Remove trailing metadata ---

    # "Grok Imagine" + anything after
    text = re.sub(r"\n\n?Grok\s*Imagine.*$", "", text, flags=re.DOTALL)

    # "Next I converted this image to a video." and similar
    text = re.sub(r"\n\n?Next I converted[^\n]*$", "", text)

    # Trailing hashtag blocks (lines of mostly hashtags)
    text = re.sub(r"\n+(?:#\w+[\s,]*)+\s*$", "", text)

    # Inline "... #aiart https://t.co/..." at end
    text = re.sub(r"\s*#\w+\s+https?://t\.co/\w+\s*$", "", text)

    # Standalone t.co URLs anywhere
    text = re.sub(r"\s*https?://t\.co/\w+", "", text)

    # "... See alt text for more"
    text = re.sub(r"\.\.\.\s*See alt text for more", "...", text)

    # Trailing single emojis
    text = re.sub(r"\s*[\U0001f300-\U0001f9ff]\s*$", "", text)

    # Leading single emojis on their own line
    text = re.sub(r"^[\U0001f300-\U0001f9ff]+\s*\n\n?", "", text, flags=re.MULTILINE)

    # "ar 5:4" aspect ratio hints at very end
    text = re.sub(r"\s*ar\s+\d+:\d+\s*$", "", text, flags=re.IGNORECASE)

    # --- Strategy 4: Handle quoted prompts ---
    stripped = text.strip()
    # Smart quotes "..."
    if stripped.startswith("\u201c") and stripped.endswith("\u201d") or stripped.startswith('"') and stripped.endswith('"') and stripped.count('"') == 2:
        inner = stripped[1:-1].strip()
        if len(inner) > 50:
            text = inner

    # --- Strategy 5: Final cleanup ---
    text = re.sub(r"\n{3,}", "\n\n", text)  # collapse blank lines
    text = text.strip()

    # If cleaning removed everything meaningful, fall back to original
    if len(text) < 20:
        return original

    return text


def main():
    cleaned_count = 0
    unchanged_count = 0

    for tmpl in SYSTEM_TEMPLATES:
        original = tmpl["prompt_text"]
        cleaned_text = clean_prompt(original)
        if cleaned_text != original.strip():
            cleaned_count += 1
            if "--verbose" in sys.argv or "-v" in sys.argv:
                print(f"--- {tmpl['id']}: {tmpl['title']}")
                print(f"  BEFORE: {original[:120]}...")
                print(f"  AFTER:  {cleaned_text[:120]}...")
                print()
        else:
            unchanged_count += 1

    print(f"Total: {len(SYSTEM_TEMPLATES)} templates")
    print(f"Cleaned: {cleaned_count}")
    print(f"Unchanged: {unchanged_count}")

    if "--apply" in sys.argv:
        filepath = (
            Path(__file__).resolve().parent.parent
            / "backend/src/pictactic/db/system_templates.py"
        )
        content = filepath.read_text(encoding="utf-8")

        for tmpl in SYSTEM_TEMPLATES:
            original = tmpl["prompt_text"]
            cleaned_text = clean_prompt(original)
            if cleaned_text != original.strip():
                # Use repr to get the escaped string representation
                old_repr = repr(original)[1:-1]
                new_repr = repr(cleaned_text)[1:-1]
                content = content.replace(old_repr, new_repr)

        filepath.write_text(content, encoding="utf-8")
        print(f"\nApplied changes to {filepath}")


if __name__ == "__main__":
    main()
