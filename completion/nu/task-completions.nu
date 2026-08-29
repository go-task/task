# Thin wrapper around `task __complete`: all suggestion logic lives in the Go engine.

# The `{completions, options}` record documented for `def` completers is
# rejected for an external one: return records or null, nothing else.
def task-external-completer [spans: list<string>] {
  let exe = ($env.TASK_EXE? | default "task")

  # The trailing empty word tells the engine the cursor is on a fresh word.
  let words = ($spans | skip 1)
  let args = (if ($words | is-empty) { [""] } else { $words })
  let current = ($args | last)

  # `complete` keeps stderr off the prompt; a missing binary raises, hence `try`.
  let result = (try { do { ^$exe "__complete" ...$args } | complete } catch { null })
  if ($result | is-empty) or $result.exit_code != 0 {
    return null
  }

  let lines = ($result.stdout | lines)
  let last = ($lines | last)
  # Protocol violation: offer nothing rather than garbage.
  if ($last | is-empty) or (not ($last | str starts-with ":")) {
    return null
  }
  let directive = (try { $last | str substring 1.. | into int } catch { 0 })
  let data = ($lines | drop 1)

  # Completion directives, mirroring internal/complete/complete.go. NoSpace (2)
  # and KeepOrder (32) need none: no space is appended, order is kept.
  let no_file_comp = (($directive | bits and 4) != 0)
  let filter_file_ext = (($directive | bits and 8) != 0)
  let filter_dirs = (($directive | bits and 16) != 0)

  # Nushell replaces the whole token, so an inline `--flag=` must be re-applied.
  let inline = ($current | parse --regex '^(?<flag>--?[^=]+=)(?<path>.*)$')
  let flag_prefix = (if ($inline | is-empty) { "" } else { $inline.0.flag })
  let path_arg = (if ($inline | is-empty) { $current } else { $inline.0.path })

  if $filter_file_ext or $filter_dirs {
    # `into glob` turns the literal path into a pattern; matching nothing raises.
    let entries = (try { ls ($"($path_arg)*" | into glob) } catch { [] })
    let matched = (if $filter_file_ext {
      $entries | where {|entry| $entry.type == "dir" or ($entry.name | path parse | get extension) in $data }
    } else {
      $entries | where type == "dir"
    })
    return ($matched | each {|entry|
      # Without a trailing separator a second <tab> matches the dir again.
      let name = (if $entry.type == "dir" { $"($entry.name)(char path_sep)" } else { $entry.name })
      { value: $"($flag_prefix)($name)" }
    })
  }

  # Nushell does not filter an external completer's results.
  let candidates = ($data
    | each {|line|
      let parts = ($line | split row --number 2 "\t")
      let value = ($parts | first)
      if ($parts | length) > 1 { { value: $value, description: ($parts | last) } } else { { value: $value } }
    }
    | where {|candidate| $candidate.value | str starts-with --ignore-case $current })

  if ($candidates | is-empty) and (not $no_file_comp) {
    return null
  }

  $candidates
}

# Nushell shares one external completer between every command, so chain to the
# installed one instead of breaking every other tool.
let task_previous_completer = ($env.config.completions.external.completer? | default null)

$env.config.completions.external.completer = {|spans|
  let exe = ($env.TASK_EXE? | default "task")
  # Compare basenames so `./task`, `/usr/local/bin/task` and `task.exe` match.
  let head = ($spans | first | path basename | str replace --regex '(?i)\.exe$' '')
  let name = ($exe | path basename | str replace --regex '(?i)\.exe$' '')
  if $head == $name {
    task-external-completer $spans
  } else if $task_previous_completer != null {
    do $task_previous_completer $spans
  } else {
    null
  }
}
