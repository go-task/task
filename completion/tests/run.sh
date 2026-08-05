#!/usr/bin/env bash
# Runs the completion test suite: builds the task binary, creates a fixture
# Taskfile with sample files and directories, then exercises the engine and
# every installed shell wrapper. Skips shells that are not installed.
set -u

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
root=$(cd "$here/../.." && pwd)

# Temp dirs for the binary and the fixture; removed on exit (including on early
# failure via the trap).
bindir=$(mktemp -d)
fixture=$(mktemp -d)
trap 'rm -rf "$bindir" "$fixture"' EXIT

# Build the binary under test.
if ! go build -o "$bindir/task" "$root/cmd/task"; then
  echo "failed to build task binary" >&2
  exit 1
fi
export TASK_BIN="$bindir/task"
# fish and PowerShell register completion for the command name `task`, so make
# `task` on PATH resolve to the binary under test.
export PATH="$bindir:$PATH"

# Fixture: a Taskfile plus files/dirs so file/dir completion has real entries.
cat > "$fixture/Taskfile.yml" <<'YML'
version: '3'

tasks:
  build:
    desc: Build it
  deploy:
    desc: Deploy it
    aliases: [dep]
    requires:
      vars:
        - name: ENV
          enum: [dev, prod]
        - REGION
  docs:serve:
    desc: Serve docs
YML
touch "$fixture/extra.yaml" "$fixture/notes.txt"
mkdir -p "$fixture/sub" "$fixture/other"
# A file inside sub/ so nested-path completion (keeping the dir prefix) is tested.
touch "$fixture/sub/nested.yml"
export TASK_FIXTURE="$fixture"

# In strict mode (set TASK_COMPLETION_STRICT=1, used in CI) a missing shell is
# a failure instead of a skip, so we never get a false pass when a shell the
# environment was expected to provide (e.g. pwsh on CI runners) is absent.
strict=${TASK_COMPLETION_STRICT:-}

fails=0
run() { # LABEL COMMAND...
  echo "== $1 =="
  if "${@:2}"; then :; else fails=$((fails + 1)); fi
  echo
}
skip() { # LABEL
  if [[ -n "$strict" ]]; then
    echo "== $1 == (MISSING — required under TASK_COMPLETION_STRICT)"
    fails=$((fails + 1))
  else
    echo "== $1 == (skipped: not installed)"
  fi
  echo
}

# The engine/protocol itself is covered by the Go tests (completion/protocol_test.go
# and internal/complete); these smokes only check how each shell wrapper
# interprets the directive.
run "bash wrapper"  bash "$here/wrapper.bash"

if command -v zsh >/dev/null 2>&1; then
  run "zsh wrapper" zsh "$here/wrapper.zsh"
else
  skip "zsh wrapper"
fi

if command -v fish >/dev/null 2>&1; then
  run "fish wrapper" fish "$here/wrapper.fish"
else
  skip "fish wrapper"
fi

pwsh_bin=$(command -v pwsh || command -v pwsh-preview || true)
if [[ -n "$pwsh_bin" ]]; then
  run "powershell wrapper" "$pwsh_bin" -NoProfile -File "$here/wrapper.ps1"
else
  skip "powershell wrapper"
fi

if ((fails)); then
  echo "completion tests: $fails suite(s) failed"
  exit 1
fi
echo "completion tests: all suites passed"
