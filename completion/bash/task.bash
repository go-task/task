#!/bin/bash

GO_TASK_PROGNAME=task

_go_task_completion()
{
  local cur
  _get_comp_words_by_ref -n : cur

  case "$cur" in
  --*)
    local options
    options="$(_parse_help task)"
    mapfile -t COMPREPLY < <(compgen -W "$options" -- "$cur")
    ;;
  *)
    local tasks
    tasks="$($GO_TASK_PROGNAME --list-all 2> /dev/null | awk 'NR>1 { sub(/:$/,"",$2); print $2 }')"
    mapfile -t COMPREPLY < <(compgen -W "$tasks" -- "$cur")
    ;;
  esac

  __ltrim_colon_completions "$cur"
}

complete -F _go_task_completion $GO_TASK_PROGNAME
