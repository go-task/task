# /bin/bash

_task_completion()
{
  local cur
  _get_comp_words_by_ref -n : cur

  case "$cur" in
  --*)
    local options="$(_parse_help task)"
    COMPREPLY=( $(compgen -W "$options" -- "$cur") )
    ;;
  *)
    local tasks="$(task --list-all | awk 'NR>1 { sub(/:$/,"",$2); print $2 }')"
    COMPREPLY=( $(compgen -W "$tasks" -- "$cur") )
    ;;
  esac

  __ltrim_colon_completions "$cur"
}

complete -F _task_completion task
