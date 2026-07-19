#!/usr/bin/env bash
# Smoke-tests how the bash wrapper routes each directive by stubbing the
# bash-completion helpers (_filedir / compopt / …) and asserting what it calls.
# Suggestion logic lives in the Go tests. Requires TASK_BIN and TASK_FIXTURE.
set -u

: "${TASK_BIN:?}"; : "${TASK_FIXTURE:?}"
export TASK_EXE="$TASK_BIN"
cd "$TASK_FIXTURE" || exit 1

fails=0
CAP=""

# Stubs standing in for the bash-completion runtime.
_init_completion() {
  words=("${TEST_WORDS[@]}")
  cword=$TEST_CWORD
  cur="${TEST_WORDS[$TEST_CWORD]}"
  prev="${TEST_WORDS[$((TEST_CWORD - 1))]}"
  return 0
}
# Records the extension arg and the value of $cur it was called with, so tests
# can assert the inline `--flag=` prefix was stripped before file completion.
_filedir() { CAP+="filedir:$* cur=$cur"$'\n'; }
compopt() { CAP+="compopt:$*"$'\n'; }
__ltrim_colon_completions() { :; }

source "$(dirname "${BASH_SOURCE[0]}")/../next/bash/task.bash"

run() {
  CAP=""
  TEST_WORDS=("$@")
  TEST_CWORD=$((${#TEST_WORDS[@]} - 1))
  COMPREPLY=()
  _task
}

reply_has() { # LABEL VALUE
  local v
  for v in "${COMPREPLY[@]}"; do [[ "$v" == "$2" ]] && { echo "  ok   $1"; return; }; done
  echo "  FAIL $1 — '$2' missing from COMPREPLY: ${COMPREPLY[*]}"
  fails=$((fails + 1))
}
cap_has() { # LABEL PATTERN
  if [[ "$CAP" == *"$2"* ]]; then echo "  ok   $1"; else
    echo "  FAIL $1 — expected '$2' in: $CAP"; fails=$((fails + 1)); fi
}
cap_hasnot() { # LABEL PATTERN
  if [[ "$CAP" == *"$2"* ]]; then
    echo "  FAIL $1 — '$2' should be absent in: $CAP"; fails=$((fails + 1)); else
    echo "  ok   $1"; fi
}

echo "bash: :4 (NoFileComp) forwards candidates, no file fallback"
run task ''
reply_has  "candidate forwarded" build
cap_hasnot "no file fallback"     "filedir:"

echo "bash: :2 (NoSpace) disables the trailing space"
run task deploy ''
cap_has    "nospace applied"      "compopt:-o nospace"

echo "bash: :8 (FilterFileExt) routes to extension-filtered files"
run task --taskfile ''
cap_has    "filedir ext glob"     "filedir:@(yml|yaml)"

echo "bash: :16 (FilterDirs) routes to directory completion"
run task --dir ''
cap_has    "filedir -d"           "filedir:-d"

echo "bash: :0 (Default) falls back to files"
run task build -- ''
cap_has    "filedir default"      "filedir:"

echo "bash: inline --flag= strips the prefix before file completion"
run task --taskfile=sub/x
cap_has    "inline cur stripped"  "cur=sub/x"

if ((fails)); then
  echo "bash: $fails failure(s)"
  exit 1
fi
echo "bash: all passed"
