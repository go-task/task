# vim: set tabstop=2 shiftwidth=2 expandtab:

_GO_TASK_COMPLETION_LIST_OPTION='--list-all'

function _task()
{
  local cur prev words cword
  _init_completion -n : || return

  # Handle special arguments of options.
  case "$prev" in
    -d|--dir)
      _filedir -d
      return $?
    ;;
    -t|--taskfile)
      _filedir yaml || return $?
      _filedir yml
      return $?
    ;;
    -o|--output)
      COMPREPLY=( $( compgen -W "interleaved group prefixed" -- $cur ) )
      return 0
    ;;
  esac

  # Handle normal options.
  case "$cur" in
    -*)
      COMPREPLY=( $( compgen -W "$(_parse_help $1)" -- $cur ) )
      return 0
    ;;
  esac

  # Get task names.
  local line tasks=()
  while read line; do
    if [[ "${line}" =~ ^\*[[:space:]]+([[:alnum:]_:]+): ]]; then
      tasks+=( ${BASH_REMATCH[1]} )
    fi
  done < <("${COMP_WORDS[@]}" $_GO_TASK_COMPLETION_LIST_OPTION 2> /dev/null)

  # Prepare task completions and post-process due to colons.
  COMPREPLY=( $( compgen -W "${tasks[*]}" -- "$cur" ) )
  __ltrim_colon_completions "$cur"
}

complete -F _task task
