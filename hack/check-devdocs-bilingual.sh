#!/usr/bin/env bash
set -euo pipefail

root="devdocs"
errors=0

if [[ ! -d "$root" ]]; then
  echo "devdocs directory does not exist; skipping bilingual check"
  exit 0
fi

while IFS= read -r english; do
  chinese="${english%.md}.zh-CN.md"
  if [[ ! -f "$chinese" ]]; then
    echo "missing Chinese companion: $chinese (for $english)" >&2
    errors=$((errors + 1))
    continue
  fi

  base="$(basename "$english")"
  if [[ "$base" =~ ^[0-9][0-9]- ]]; then
    english_sections="$(grep -E '^## [0-9]+\.' "$english" | sed -E 's/^## ([0-9]+)\..*/\1/' | paste -sd, - || true)"
    chinese_sections="$(grep -E '^## [0-9]+\.' "$chinese" | sed -E 's/^## ([0-9]+)\..*/\1/' | paste -sd, - || true)"
    if [[ "$english_sections" != "$chinese_sections" ]]; then
      echo "numbered section mismatch:" >&2
      echo "  English: $english -> [$english_sections]" >&2
      echo "  Chinese: $chinese -> [$chinese_sections]" >&2
      errors=$((errors + 1))
    fi
  fi
done < <(find "$root" -type f -name '*.md' ! -name '*.zh-CN.md' | sort)

while IFS= read -r chinese; do
  english="${chinese%.zh-CN.md}.md"
  if [[ ! -f "$english" ]]; then
    echo "orphan Chinese document: $chinese (missing $english)" >&2
    errors=$((errors + 1))
  fi
done < <(find "$root" -type f -name '*.zh-CN.md' | sort)

if (( errors > 0 )); then
  echo "bilingual devdocs validation failed with $errors error(s)" >&2
  exit 1
fi

echo "bilingual devdocs validation passed"
