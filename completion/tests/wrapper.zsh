#!/usr/bin/env zsh
# Smoke-tests how the zsh wrapper routes each directive by stubbing the
# completion functions (_describe / _files / _path_files) and asserting what it
# calls. Suggestion logic lives in the Go tests. Requires TASK_BIN, TASK_FIXTURE.

export TASK_EXE=$TASK_BIN
cd $TASK_FIXTURE

integer fails=0
local CAP
compdef() { }   # no-op: we call _task directly, not through compinit

_describe() {
    local arr=$4
    CAP+="describe_opts:${@[5,-1]}"$'\n'
    local c; for c in ${(P)arr}; do CAP+="cand:$c"$'\n'; done
}
_files()      { CAP+="files:$*"$'\n' }
_path_files() { CAP+="path_files:$*"$'\n' }

# Sourcing (not autoloading) defines _task and avoids the autoload first-call
# quirk; the trailing `compdef` call is stubbed above.
source ${0:A:h}/../next/zsh/_task

run() {
    CAP=""
    local -a words=("$@")
    integer CURRENT=$#words
    local curcontext=":completion:complete:task:"
    _task
}

has() { # LABEL PATTERN
    if [[ "$CAP" == *"$2"* ]]; then
        echo "  ok   $1"
    else
        echo "  FAIL $1 — expected '$2' in:"$'\n'"$CAP"
        (( fails++ ))
    fi
}
hasnot() { # LABEL PATTERN
    if [[ "$CAP" == *"$2"* ]]; then
        echo "  FAIL $1 — '$2' should be absent in:"$'\n'"$CAP"
        (( fails++ ))
    else
        echo "  ok   $1"
    fi
}

echo "zsh: :4 (NoFileComp) forwards candidates, no file fallback"
run task ''
has    "candidate forwarded"  "cand:build"
hasnot "no file fallback"     "files:"

echo "zsh: :2|:32 (NoSpace|KeepOrder) map to -S and -V"
run task deploy ''
has    "NoSpace -> -S"         "describe_opts:-S"
has    "KeepOrder -> -V"       "-V"

echo "zsh: :8 (FilterFileExt) routes to extension-filtered files"
run task --taskfile ''
has    "files glob"            "files:"
has    "yml in glob"           "yml"

echo "zsh: :16 (FilterDirs) routes to directory completion"
run task --dir ''
has    "path_files -/"         "path_files:-/"

echo "zsh: :0 (Default) falls back to files"
run task build -- ''
has    "files default"         "files:"

if (( fails )); then
    echo "zsh: $fails failure(s)"
    exit 1
fi
echo "zsh: all passed"
