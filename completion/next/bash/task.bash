# vim: set tabstop=2 shiftwidth=2 expandtab:
#
# Thin wrapper around `task __complete`: all suggestion logic lives in the Go engine.

TASK_CMD="${TASK_EXE:-task}"

# `=` stays inside the current word (see `_init_completion -n =:`), so an inline
# `--flag=` prefix must be stripped before _filedir and re-applied after.
_task_filedir() {
  local fpfx="" savecur="$cur"
  if [[ "$cur" == -*=* ]]; then
    fpfx="${cur%%=*}="
    cur="${cur#*=}"
  fi
  _filedir ${1:+"$1"}
  cur="$savecur"
  if [[ -n "$fpfx" ]]; then
    COMPREPLY=( ${COMPREPLY[@]+"${COMPREPLY[@]/#/$fpfx}"} )
  fi
}

_task() {
  local cur prev words cword

  # Completion directives, mirroring internal/complete/complete.go.
  local -ri NO_SPACE=2 NO_FILE_COMP=4 FILTER_FILE_EXT=8 FILTER_DIRS=16 KEEP_ORDER=32

  # `=` and `:` out of the word breaks: `--output=`, `docs:serve` stay one token.
  _init_completion -n =: || return

  local -a args=( "${words[@]:1:cword}" )
  if (( ${#args[@]} == 0 )); then
    args=( "" )
  fi

  local output
  output=$("$TASK_CMD" __complete "${args[@]}" 2>/dev/null)
  if [[ -z "$output" ]]; then
    _task_filedir
    return
  fi

  local -a lines=()
  local line
  while IFS= read -r line; do
    lines+=( "$line" )
  done <<< "$output"

  local last_idx=$(( ${#lines[@]} - 1 ))
  local directive="${lines[$last_idx]#:}"
  unset 'lines[$last_idx]'

  if (( directive & FILTER_FILE_EXT )); then
    local exts=""
    # ${arr[@]+…} guards an empty array under `set -u` in bash 3.2 (macOS).
    for line in ${lines[@]+"${lines[@]}"}; do
      exts+="${exts:+|}$line"
    done
    _task_filedir "@($exts)"
    return
  fi

  if (( directive & FILTER_DIRS )); then
    _task_filedir -d
    return
  fi

  # Not `compgen -W`: it splits the word list on IFS, mangling values with spaces.
  local value
  COMPREPLY=()
  for line in ${lines[@]+"${lines[@]}"}; do
    value="${line%%$'\t'*}"
    if [[ -z "$cur" || "$value" == "$cur"* ]]; then
      COMPREPLY+=( "$value" )
    fi
  done

  if (( directive & NO_SPACE )); then
    compopt -o nospace 2>/dev/null
  fi

  # nosort needs bash 4.4; the 3.2 shipped by macOS ignores it and stays sorted.
  if (( directive & KEEP_ORDER )); then
    compopt -o nosort 2>/dev/null
  fi

  __ltrim_colon_completions "$cur"

  if (( ${#COMPREPLY[@]} == 0 )) && ! (( directive & NO_FILE_COMP )); then
    _task_filedir
  fi
}

complete -F _task "$TASK_CMD"
