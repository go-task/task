#!/usr/bin/env nu
# Smoke-tests how the Nushell wrapper routes each directive. External completers
# only run in the interactive REPL, so the closure is called directly.
# Set up by run.sh: $env.TASK_FIXTURE, and `task` on PATH = the binary under test.

# `source` needs a parse-time constant path.
const TASK_NU = (path self "../nu/task-completions.nu")

# Installed before the wrapper is sourced, to assert the delegation path.
$env.config.completions.external.completer = {|spans| [{ value: $"prev:($spans | first)" }] }

source $TASK_NU

cd $env.TASK_FIXTURE

let completer = $env.config.completions.external.completer

def cands [spans: list<string>] {
  let out = (do $completer $spans)
  if $out == null { [] } else { $out | get value }
}

def has [label: string, spans: list<string>, value: string] {
  let values = (cands $spans)
  if $value in $values {
    print $"  ok   ($label)"
    0
  } else {
    print $"  FAIL ($label) — '($value)' missing from: ($values | str join ' ')"
    1
  }
}

def hasnot [label: string, spans: list<string>, value: string] {
  if $value in (cands $spans) {
    print $"  FAIL ($label) — '($value)' should be absent"
    1
  } else {
    print $"  ok   ($label)"
    0
  }
}

def check [label: string, ok: bool] {
  if $ok {
    print $"  ok   ($label)"
    0
  } else {
    print $"  FAIL ($label)"
    1
  }
}

mut fails = 0

print "nu: :4 (NoFileComp) forwards candidates, offers no files"
$fails += (has    "candidate forwarded" [task ""] "build")
$fails += (hasnot "no file fallback"    [task ""] "notes.txt")

print "nu: filters candidates by the current word"
$fails += (has    "prefix keeps match"  [task b] "build")
$fails += (hasnot "prefix drops others" [task b] "deploy")

print "nu: :16 (FilterDirs) offers directories only"
$fails += (has    "dir offered"         [task --dir ""] $"sub(char path_sep)")
$fails += (hasnot "no plain file"       [task --dir ""] "notes.txt")

print "nu: :8 (FilterFileExt) filters by extension"
$fails += (has    "matching file"       [task --taskfile ""] "Taskfile.yml")
$fails += (hasnot "non-matching file"   [task --taskfile ""] "notes.txt")

print "nu: nested path completion keeps the directory prefix"
$fails += (has    "prefix kept"         [task --taskfile $"sub(char path_sep)"] $"sub(char path_sep)nested.yml")

print "nu: inline --flag=path keeps the --flag= prefix"
$fails += (has    "inline nested"       [task $"--taskfile=sub(char path_sep)"] $"--taskfile=sub(char path_sep)nested.yml")
$fails += (hasnot "inline non-matching" [task "--taskfile="] "--taskfile=notes.txt")

print "nu: :2|:32 (NoSpace|KeepOrder) keep the order the engine emitted"
let vars = (cands [task deploy ""])
$fails += (has   "required var offered" [task deploy ""] "ENV=dev")
$fails += (check "declaration order kept" (($vars | enumerate | where item == "ENV=dev" | get 0.index) < ($vars | enumerate | where item == "REGION=" | get 0.index)))

print "nu: :0 (Default) returns null so Nushell completes files itself"
$fails += (check "null returned" ((do $completer [task build "--" ""]) == null))

print "nu: other commands go to the previously installed completer"
$fails += (has   "delegated"            [git status ""] "prev:git")

if $fails != 0 {
  print $"nu: ($fails) failure\(s\)"
  exit 1
}
print "nu: all passed"
