#!/usr/bin/env bash
#
# The website builds two channels: `website/src/docs` documents the upcoming
# release, `website/src/versioned/docs` documents the released version served on
# taskfile.dev. Fixing the published docs is allowed, but the fix has to land in
# both places -- cmd/release overwrites the versioned copy at every release, so
# a fix applied only there is silently lost.
#
# Only modifications are checked. Added files are a deliberate promotion (or the
# initial bootstrap) and deleted files cannot be lost.

set -euo pipefail

base="${1:-}"
if [[ -z "$base" ]]; then
  echo "usage: $0 <base-sha>" >&2
  exit 2
fi

next_dir="website/src/docs"
versioned_dir="website/src/versioned/docs"

missing=()
while IFS= read -r file; do
  [[ -z "$file" ]] && continue
  counterpart="$next_dir/${file#"$versioned_dir/"}"
  if git diff --quiet "$base" HEAD -- "$counterpart"; then
    missing+=("$file")
  fi
done < <(git diff --name-status --diff-filter=MR "$base" HEAD -- "$versioned_dir" | cut -f2)

if [[ ${#missing[@]} -eq 0 ]]; then
  echo "Versioned docs are consistent with the next docs."
  exit 0
fi

echo "The following files were changed in $versioned_dir without the matching" >&2
echo "change in $next_dir:" >&2
echo >&2
for file in "${missing[@]}"; do
  echo "  $file" >&2
  echo "    -> also update $next_dir/${file#"$versioned_dir/"}" >&2
done
echo >&2
echo "cmd/release overwrites $versioned_dir at every release, so a fix applied" >&2
echo "only to the published docs would be lost. Port it to $next_dir as well." >&2
exit 1
