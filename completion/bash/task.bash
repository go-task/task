# vim: set tabstop=2 shiftwidth=2 expandtab:
#
# Thin wrapper around `task __complete`. All suggestion logic lives in the
# Go engine — do not add completion logic here.

TASK_CMD="${TASK_EXE:-task}"

_task() {
  local cur prev words cword
  # Exclude both `=` and `:` from the word breaks so `--output=` and
  # `docs:serve` reach the engine as single tokens.
  _init_completion -n =: || return

  local -a args=()
  if (( cword > 0 )); then
    args=( "${words[@]:1:cword}" )
  fi
  if (( ${#args[@]} == 0 )); then
    args=( "" )
  fi

  local output
  output=$("$TASK_CMD" __complete "${args[@]}" 2>/dev/null)
  if [[ -z "$output" ]]; then
    _filedir
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

  if (( directive & 8 )); then
    local exts=""
    for line in "${lines[@]}"; do
      exts+="${exts:+|}$line"
    done
    _filedir "@($exts)"
    return
  fi

  if (( directive & 16 )); then
    _filedir -d
    return
  fi

  # Prefix-filter by hand instead of `compgen -W`: the latter joins/splits the
  # word list on IFS, which mangles any suggestion value containing a space.
  local value
  COMPREPLY=()
  for line in "${lines[@]}"; do
    value="${line%%$'\t'*}"
    if [[ -z "$cur" || "$value" == "$cur"* ]]; then
      COMPREPLY+=( "$value" )
    fi
  done

  if (( directive & 2 )); then
    compopt -o nospace 2>/dev/null
  fi

  __ltrim_colon_completions "$cur"

  if (( ${#COMPREPLY[@]} == 0 )) && ! (( directive & 4 )); then
    _filedir
  fi
}

complete -F _task "$TASK_CMD"
