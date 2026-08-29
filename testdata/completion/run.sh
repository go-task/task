#!/usr/bin/env bash
# Builds the task binary and a fixture Taskfile, then runs every installed shell
# wrapper against them. The engine itself is covered by the Go tests.
set -u

# fish, Nushell and PowerShell resolve the binary through these; an ambient value
# would silently test something other than the binary built below.
unset TASK_EXE GO_TASK_PROGNAME

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
root=$(cd "$here/../.." && pwd)

bindir=$(mktemp -d)
fixture=$(mktemp -d)
trap 'rm -rf "$bindir" "$fixture"' EXIT

if ! go build -o "$bindir/task" "$root/cmd/task"; then
  echo "failed to build task binary" >&2
  exit 1
fi
export TASK_BIN="$bindir/task"
# fish and PowerShell register completion for the command name `task`.
export PATH="$bindir:$PATH"

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
# Nested path completion must keep the directory prefix.
touch "$fixture/sub/nested.yml"
# Shells must pass a quoted `--dir` value to the engine unquoted, and quote it
# back on insert.
mkdir -p "$fixture/with space"
cat > "$fixture/with space/Taskfile.yml" <<'YML'
version: '3'

tasks:
  spaced:
    desc: Task from the spaced dir
YML
export TASK_FIXTURE="$fixture"

# Strict mode (CI) turns a missing shell into a failure instead of a skip, so an
# absent pwsh never reads as a pass.
strict=${TASK_COMPLETION_STRICT:-}

fails=0
run() { # LABEL COMMAND...
  echo "== $1 =="
  "${@:2}" || fails=$((fails + 1))
  echo
}
run_if() { # BIN LABEL COMMAND...
  if command -v "$1" >/dev/null 2>&1; then run "${@:2}"; else skip "$2"; fi
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

run    "bash wrapper" bash "$here/wrapper.bash"
run_if zsh  "zsh wrapper"  zsh  "$here/wrapper.zsh"
run_if fish "fish wrapper" fish "$here/wrapper.fish"
# --no-config-file: the user's own external completer must not interfere.
run_if nu   "nu wrapper"   nu --no-config-file "$here/wrapper.nu"

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
