#!/usr/bin/env fish
# Smoke-tests how the fish wrapper routes each directive, via `complete -C`
# (real completions, no TTY). Suggestion logic lives in the Go tests.
# Set up by run.sh: TASK_FIXTURE, and `task` on PATH = the binary under test.

cd $TASK_FIXTURE
source (dirname (status -f))/../next/fish/task.fish

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
