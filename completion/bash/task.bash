# vim: set tabstop=2 shiftwidth=2 expandtab:

_GO_TASK_COMPLETION_LIST_OPTION='--list-all'

function _task()
{
  local cur prev words cword
  _init_completion -n : || return

  # Check for `--` within command-line and quit or strip suffix.
  local i
  for i in "${!words[@]}"; do
    if [ "${words[$i]}" == "--" ]; then
      # Do not complete words following `--` which are passed to CLI_ARGS.
      [ $cword -gt $i ] && return
      # Remove the words following `--` to not put --list in CLI_ARGS.
      words=( "${words[@]:0:$i}" )
      break
    fi
  done

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

  # Obtain task names.
  local tasks=( $( "${words[@]}" --silent $_GO_TASK_COMPLETION_LIST_OPTION 2> /dev/null ) )

  # Obtain alias names. (TODO: Include aliases in --list --silent as well.)
  local line
  while read -r line; do
    if [[ "${line}" =~ \(aliases:[[:space:]]+(.+)\) ]]; then
      tasks+=( ${BASH_REMATCH[1]} )
    fi
  done < <( "${words[@]}" $_GO_TASK_COMPLETION_LIST_OPTION 2> /dev/null )

  # Prepare task name proposals.
  COMPREPLY=( $( compgen -W "${tasks[*]%,}" -- "$cur" ) )

  # Post-process proposals because task names might contain colons.
  __ltrim_colon_completions "$cur"
}

complete -F _task task
