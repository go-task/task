# Thin wrapper around `task __complete`. All suggestion logic lives in the
# Go engine — do not add completion logic here.

set -l GO_TASK_PROGNAME (if set -q GO_TASK_PROGNAME; echo $GO_TASK_PROGNAME; else if set -q TASK_EXE; echo $TASK_EXE; else; echo task; end)

# Completion directives, mirroring internal/complete/complete.go. fish's `math`
# has no bitwise operators, so bits are stored as their power-of-two value and
# tested with integer division + modulo via __task_test_bit.
set -g __task_directive_no_space 2
set -g __task_directive_no_file_comp 4
set -g __task_directive_filter_file_ext 8
set -g __task_directive_filter_dirs 16
set -g __task_directive_keep_order 32

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

  # The main completion is registered with `--no-files`, which disables fish's
  # native file fallback. Every file-completion directive must therefore be
  # served here, otherwise nothing is offered (e.g. `--cacert`, after `--`).

  # FilterFileExt: the engine emits the allowed extensions as the data lines.
  # __fish_complete_suffix only *prioritizes* the extension, so filter the file
  # list ourselves — keeping directories so the user can still descend into them.
  if __task_test_bit $directive $__task_directive_filter_file_ext
    for entry in (__fish_complete_path $current)
      set -l name (string split -f1 \t -- $entry)
      if string match -qr '/$' -- $name
        printf '%s\n' $entry
        continue
      end
      for ext in $data
        if string match -qr "\.$ext\$" -- $name
          printf '%s\n' $entry
          break
        end
      end
    end
    return
  end

  # FilterDirs: complete directories only.
  if __task_test_bit $directive $__task_directive_filter_dirs
    __fish_complete_directories $current
    return
  end

  # Emit the `value\tdescription` candidates (fish reads the tab as the
  # separator between the completion and its description).
  for line in $data
    printf '%s\n' $line
  end

  # NoFileComp unset → also offer files, since `--no-files` suppressed the
  # native fallback. Covers DirectiveDefault (e.g. `--cacert`, after `--`).
  if not __task_test_bit $directive $__task_directive_no_file_comp
    __fish_complete_path $current
  end
end

# Single registration: all task names, flags, flag values and file completion
# flow through the engine. `--no-files` prevents fish from mixing in files when
# the engine says not to (NoFileComp); `__task_complete` re-adds them otherwise.
complete -c $GO_TASK_PROGNAME --no-files -a "(__task_complete)"
