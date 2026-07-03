#!/usr/bin/env bash
# Tests the `task __complete` protocol directly (shell-agnostic). This is the
# backbone: it validates the candidates and directive the engine emits, which
# is what drives every shell wrapper.
#
# Requires: TASK_BIN (path to the task binary), TASK_FIXTURE (dir with a
# Taskfile.yml). Exits non-zero on the first failure.
set -u

: "${TASK_BIN:?TASK_BIN must point to the task binary}"
: "${TASK_FIXTURE:?TASK_FIXTURE must point to the fixture directory}"
cd "$TASK_FIXTURE" || exit 1

fails=0
out() { "$TASK_BIN" __complete "$@" 2>/dev/null; }
vals() { out "$@" | sed '$d' | cut -f1; }   # candidate values, sans the :N line
dirv() { out "$@" | tail -1; }              # the :N directive line

has() { # LABEL VALUE ARGS...
  local label=$1 value=$2; shift 2
  if vals "$@" | grep -qxF -- "$value"; then
    echo "  ok   $label"
  else
    echo "  FAIL $label — expected value '$value' among: $(vals "$@" | tr '\n' ' ')"
    fails=$((fails + 1))
  fi
}
hasnot() { # LABEL VALUE ARGS...
  local label=$1 value=$2; shift 2
  if vals "$@" | grep -qxF -- "$value"; then
    echo "  FAIL $label — value '$value' should be absent"
    fails=$((fails + 1))
  else
    echo "  ok   $label"
  fi
}
directive() { # LABEL EXPECTED ARGS...
  local label=$1 expected=$2; shift 2
  local got; got=$(dirv "$@")
  if [[ "$got" == "$expected" ]]; then
    echo "  ok   $label"
  else
    echo "  FAIL $label — expected directive '$expected', got '$got'"
    fails=$((fails + 1))
  fi
}

echo "engine: task names"
has       "lists tasks"        build ''
has       "lists aliases"      dep ''
directive "NoFileComp"         ':4' ''

echo "engine: completion-control flags"
hasnot    "--no-aliases drops aliases"       dep --no-aliases ''
has       "--no-aliases keeps tasks"         deploy --no-aliases ''

echo "engine: flags"
has       "lists flags"        --taskfile -
directive "flags NoFileComp"   ':4' -

echo "engine: flag values"
has       "inline --output= is full form"    --output=interleaved --output=
directive "inline NoFileComp"                ':4' --output=
has       "separate --output is bare"        interleaved --output ''
has       "--sort values"                    alphanumeric --sort ''

echo "engine: file/dir directives"
has       "--taskfile emits yml"   yml --taskfile ''
has       "--taskfile emits yaml"  yaml --taskfile ''
directive "--taskfile FilterFileExt" ':8' --taskfile ''
directive "--dir FilterDirs"         ':16' --dir ''

echo "engine: task variables"
has       "required var with enum"   ENV=dev deploy ''
has       "required var without enum" REGION= deploy ''
directive "vars NoSpace|NoFileComp|KeepOrder" ':38' deploy ''

echo "engine: after --"
directive "after -- is default"      ':0' deploy -- ''

if (( fails )); then
  echo "engine: $fails failure(s)"
  exit 1
fi
echo "engine: all passed"
