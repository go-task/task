#!/usr/bin/env fish
# Tests the fish wrapper end-to-end via `complete -C`, which asks fish for the
# real completions of a command line without a TTY. The `task` command must
# resolve to the binary under test (run.sh puts a symlink on PATH).
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

echo "fish: task names (no files)"
has    "lists tasks"        'task ' build
has    "lists aliases"      'task ' dep
hasnot "no files for tasks" 'task ' notes.txt

echo "fish: task variables"
has    "required vars"      'task deploy ' ENV=dev

echo "fish: flag values"
has    "enum values"        'task --output ' interleaved

echo "fish: --dir completes directories only"
has    "dirs offered"       'task --dir ' sub/
hasnot "no plain files"     'task --dir ' notes.txt

echo "fish: --taskfile filters by extension"
has    "yaml offered"       'task --taskfile ' Taskfile.yml
hasnot "txt filtered out"   'task --taskfile ' notes.txt

echo "fish: after -- completes files"
has    "files after --"     'task build -- ' notes.txt

if test $fails -ne 0
    echo "fish: $fails failure(s)"
    exit 1
end
echo "fish: all passed"
