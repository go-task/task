#!/usr/bin/env bash
# Tests the bash wrapper by stubbing the bash-completion helpers it calls
# (_init_completion / _filedir / compopt / __ltrim_colon_completions) and
# asserting the resulting COMPREPLY and file routing. Deterministic, no TTY,
# and works without the bash-completion package installed.
#
# Requires: TASK_BIN (task binary), TASK_FIXTURE (dir with a Taskfile.yml).
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
_filedir() { CAP+="filedir:$*"$'\n'; }
compopt() { CAP+="compopt:$*"$'\n'; }
__ltrim_colon_completions() { :; }

source "$(dirname "${BASH_SOURCE[0]}")/../bash/task.bash"

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

echo "bash: task names (no file fallback)"
run task ''
reply_has  "lists tasks"       build
reply_has  "lists aliases"     dep
cap_hasnot "no file fallback"  "filedir:"

echo "bash: task variables"
run task deploy ''
reply_has  "required vars"     "ENV=dev"
cap_has    "NoSpace nospace"   "compopt:-o nospace"

echo "bash: inline --output= is full form"
run task '--output='
reply_has  "full-form value"   "--output=interleaved"

echo "bash: --dir routes to directory completion"
run task --dir ''
cap_has    "filedir -d"        "filedir:-d"

echo "bash: --taskfile routes to extension-filtered files"
run task --taskfile ''
cap_has    "filedir ext glob"  "filedir:@(yml|yaml)"

echo "bash: after -- falls back to files"
run task build -- ''
cap_has    "filedir after --"  "filedir:"

if ((fails)); then
  echo "bash: $fails failure(s)"
  exit 1
fi
echo "bash: all passed"
