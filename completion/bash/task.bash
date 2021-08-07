_task_completion()
{
  local scripts;
  local curr_arg;

  # Remove colon from word breaks
  COMP_WORDBREAKS=${COMP_WORDBREAKS//:}

  scripts=$(task -l | sed '1d' | awk '{ print $2 }' | sed 's/:$//');

  curr_arg="${COMP_WORDS[COMP_CWORD]:-"."}"

  # Do not accept more than 1 argument
  if [ "${#COMP_WORDS[@]}" != "2" ]; then
    return
  fi

  COMPREPLY=($(compgen -c | echo "$scripts" | grep -- $curr_arg));
}

complete -F _task_completion task
