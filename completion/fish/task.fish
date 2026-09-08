# Thin wrapper around `task __complete`: all suggestion logic lives in the Go engine.

set -l GO_TASK_PROGNAME (if set -q GO_TASK_PROGNAME; echo $GO_TASK_PROGNAME; else if set -q TASK_EXE; echo $TASK_EXE; else; echo task; end)

# Completion directives, mirroring internal/complete/complete.go. `math` has no
# bitwise operators, hence __task_test_bit. NoSpace (2) and KeepOrder (32) need
# none: fish appends no space and keeps the order.
set -g __task_directive_no_file_comp 4
set -g __task_directive_filter_file_ext 8
set -g __task_directive_filter_dirs 16

function __task_test_bit --argument-names value bit
  test (math "floor($value / $bit) % 2") -eq 1
end

function __task_complete --inherit-variable GO_TASK_PROGNAME
  set -l tokens (commandline -opc)
  set -l current (commandline -ct)
  set -l args
  if test (count $tokens) -gt 1
    set args $tokens[2..-1]
  end
  set args $args $current

  set -l output ($GO_TASK_PROGNAME __complete $args 2>/dev/null)
  set -l count (count $output)
  if test $count -eq 0
    return
  end

  set -l last $output[$count]
  if not string match -q ':*' -- $last
    # Protocol violation: emit raw lines as a fallback.
    printf '%s\n' $output
    return
  end

  set -l directive (string replace -r '^:' '' -- $last)
  set -l data
  if test $count -gt 1
    set data $output[1..(math $count - 1)]
  end

  # The registration below passes `--no-files`, so every file-completion
  # directive must be served here or nothing is offered at all.

  # fish replaces the whole token, so an inline `--flag=` must be kept on every
  # candidate.
  set -l flagpfx ""
  set -l pathcur $current
  if string match -qr '^--?[^=]+=' -- $current
    set flagpfx (string replace -r '=.*$' '=' -- $current)
    set pathcur (string replace -r '^--?[^=]+=' '' -- $current)
  end

  # __fish_complete_suffix prioritizes the extension instead of filtering.
  if __task_test_bit $directive $__task_directive_filter_file_ext
    for entry in (__fish_complete_path $pathcur)
      set -l name (string split -f1 \t -- $entry)
      if string match -qr '/$' -- $name
        printf '%s%s\n' $flagpfx $entry
        continue
      end
      for ext in $data
        if string match -qr "\.$ext\$" -- $name
          printf '%s%s\n' $flagpfx $entry
          break
        end
      end
    end
    return
  end

  if __task_test_bit $directive $__task_directive_filter_dirs
    for entry in (__fish_complete_directories $pathcur)
      printf '%s%s\n' $flagpfx $entry
    end
    return
  end

  for line in $data
    printf '%s\n' $line
  end

  # NoFileComp unset → offer files too (DirectiveDefault).
  if not __task_test_bit $directive $__task_directive_no_file_comp
    for entry in (__fish_complete_path $pathcur)
      printf '%s%s\n' $flagpfx $entry
    end
  end
end

# fish accumulates `complete` entries instead of replacing them, so an older
# completion would keep contributing alongside the engine.
complete -c $GO_TASK_PROGNAME -e

# `--no-files` keeps fish from mixing in files against the engine's directive.
complete -c $GO_TASK_PROGNAME --no-files -a "(__task_complete)"
