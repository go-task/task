#!/usr/bin/env bash
#
# The website builds two channels: `website/src/{docs,blog}` is what is being
# written for the upcoming release, `website/src/latest/{docs,blog}` is what
# taskfile.dev serves today. Fixing published content is allowed, but the fix
# has to land in both places -- cmd/release overwrites the latest copy at every
# release, so a fix applied only there is silently lost.
#
# Only modifications are checked. Added files are deliberate: a post added to
# `latest/blog` is published now, a post added to `blog` alone ships with the
# release. Deleted files cannot be lost.

set -euo pipefail

base="${1:-}"
if [[ -z "$base" ]]; then
  echo "usage: $0 <base-sha>" >&2
  exit 2
fi

next_dir="website/src"
latest_dir="website/src/latest"

missing=()
while IFS= read -r file; do
  [[ -z "$file" ]] && continue
  counterpart="$next_dir/${file#"$latest_dir/"}"
  if git diff --quiet "$base" HEAD -- "$counterpart"; then
    missing+=("$file")
  fi
done < <(git diff --name-status --diff-filter=MR "$base" HEAD -- "$latest_dir" | cut -f2)

if [[ ${#missing[@]} -eq 0 ]]; then
  echo "Latest content is consistent with the next content."
  exit 0
fi

echo "The following files were changed under $latest_dir without the matching" >&2
echo "change under $next_dir:" >&2
echo >&2
for file in "${missing[@]}"; do
  echo "  $file" >&2
  echo "    -> also update $next_dir/${file#"$latest_dir/"}" >&2
done
echo >&2
echo "cmd/release overwrites $latest_dir at every release, so a fix applied only" >&2
echo "to the published content would be lost. Port it to $next_dir as well." >&2
exit 1
