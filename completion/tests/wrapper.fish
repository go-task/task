#!/usr/bin/env fish
# Smoke-tests how the fish wrapper INTERPRETS each directive (files vs dirs vs
# no files) via `complete -C`, which asks fish for the real completions without
# a TTY. The suggestion logic (which tasks/aliases/vars) is covered by the Go
# tests; here we only check routing. `task` must resolve to the binary under
# test (run.sh puts a symlink on PATH).
#
# Requires: TASK_FIXTURE (dir with a Taskfile.yml and sample files/dirs).

cd $TASK_FIXTURE
source (dirname (status -f))/../fish/task.fish

set -g fails 0

function cands
    complete -C $argv[1] | string split -f1 \t
end

function has # LABEL LINE VALUE
    if contains -- $argv[3] (cands $argv[2])
        echo "  ok   $argv[1]"
    else
        echo "  FAIL $argv[1] — '$argv[3]' missing from: "(cands $argv[2])
        set fails (math $fails + 1)
    end
end

function hasnot # LABEL LINE VALUE
    if contains -- $argv[3] (cands $argv[2])
        echo "  FAIL $argv[1] — '$argv[3]' should be absent"
        set fails (math $fails + 1)
    else
        echo "  ok   $argv[1]"
    end
end

echo "fish: :4 (NoFileComp) forwards candidates, offers no files"
has    "candidate forwarded"  'task ' build
hasnot "no file fallback"     'task ' notes.txt

echo "fish: :16 (FilterDirs) offers directories only"
has    "dir offered"          'task --dir ' sub/
hasnot "no plain file"        'task --dir ' notes.txt

echo "fish: :8 (FilterFileExt) filters by extension"
has    "matching file"        'task --taskfile ' Taskfile.yml
hasnot "non-matching file"    'task --taskfile ' notes.txt

echo "fish: :0 (Default) falls back to files"
has    "file offered"         'task build -- ' notes.txt

if test $fails -ne 0
    echo "fish: $fails failure(s)"
    exit 1
end
echo "fish: all passed"
