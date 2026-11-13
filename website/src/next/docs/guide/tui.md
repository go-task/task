---
title: Interactive TUI
description:
  Browse and run tasks in a full-screen terminal interface with a task
  navigator, per-task output, and copy and save controls.
section: Guide
docType: guide
outline: deep
---

# Interactive TUI

Run `task --tui` (or `task -T`) to open an interactive, full-screen Terminal
User Interface (TUI). The launcher lists the available non-internal tasks and
their descriptions. Type to filter by task name or description and use the
up/down arrows to select a task. Press Enter to run it in the execution
dashboard, or press Ctrl+R to leave the TUI and run it with Task's normal
terminal output. Escape clears the current filter and Ctrl+C quits.

You can skip the launcher by providing task names directly:

```shell
$ task --tui build test lint
$ task --tui --parallel build test lint
```

After direct execution completes, press Escape or `b` to open the launcher.

Each requested task is displayed as an independent root. As with regular Task
invocations, multiple requested tasks run sequentially by default; pass
`--parallel` to run them concurrently.

During execution, the left pane shows a task navigator and the right pane shows
the output of the currently selected task.

The requested root task appears at the top and can be selected to inspect output
from commands that it runs directly. When the root only orchestrates other
tasks, the first child is selected automatically. By default, tasks are nested
beneath the task that invoked them. Repeated executions have separate entries,
while calls that join an existing `run: once` or `run: when_changed` execution
remain visible at each location with a `↳` marker and share the owner's status
and output. Pass `--tui-task-navigator list` to show all tasks reached from each
root in a compact, single-level list instead. Press `n` during a run to switch
between the two.

Each task shows a status icon, including distinct canceled and skipped states.
Canceled tasks were interrupted, while skipped tasks were never attempted after
an earlier sequential task failed. Pass `--tui-status labels` to replace the
icons with text labels.

Both `--tui-status` and `--tui-task-navigator` can be set as defaults in
[`.taskrc.yml`](../reference/config.md#tui).

Press `?` at any time to see every key available in the current view.

Use Tab or the left/right arrow keys to switch between the task navigator and
the output pane. Clicking either pane also focuses it.

When the navigator is focused, use the up/down arrows or `j`/`k` to select a
task. You can also click a task directly. When the output pane is focused, use
the following controls to scroll:

- Up/down arrows or `j`/`k`
- Page Up and Page Down
- `g` and `G` to jump to the beginning or end
- Mouse wheel

Output taller than the pane draws a scrollbar on the pane's right border,
showing both where you are and how much there is. The top right of the pane
shows how the selected task ended, along with the exit code when the task ran a
command that reported one. A task that failed only because one of its
dependencies did shows no code of its own.

Each task shows how long it ran, counting up while it is running and keeping its
final duration afterwards. Quick tasks are reported in milliseconds. A task that
has not started has no duration, which is not the same as a duration of zero. On
a narrow terminal the durations are dropped so that task names keep their space.

Press `f` to show the selected task's output fullscreen. Incoming output remains
visible; the view follows it while at the bottom and preserves the current
position after you scroll up. Press `f` again or Escape to return to the
two-pane view.

Fullscreen is where lines are picked out of the output. A cursor marks one line,
and the controls above move it, scrolling as needed. Press `v` (or `V`, after
Vim's visual mode) to start selecting: the lines between where you pressed it
and where the cursor is now are selected, so moving up from that point selects
upwards. Press `v` again, or Escape, to cancel the selection; a second Escape
leaves fullscreen.

The two states look different, so that being in one is never a guess. A resting
cursor is marked quietly; a live selection is drawn in the accent colour.

With lines selected, `y` and `Y` copy those lines instead of the whole output.
They are copied as they were written, so a line too long for the screen arrives
whole rather than in the pieces it was folded into. Highlighted lines are drawn
without their own colours, because a highlight cannot survive the escape
sequences inside them; the copy still carries those sequences for `Y`.

## Copying task output {#copying-task-output}

Selecting text with the mouse does not work inside the dashboard. A terminal
discards a selection whenever the screen is repainted, and scrolling either pane
is a repaint. These controls get the text out instead:

- `y` copies the selected task's output to the system clipboard with its ANSI
  escape sequences stripped, which is what a terminal gives you when you select
  text by hand.
- `Y` copies it with those sequences intact, for pasting somewhere that renders
  them, such as an editor with an ANSI extension. They carry bold, dim and
  underline as well as colour.
- `s` saves the selected task's output, and `S` saves every task's output to a
  folder, one file per task. Both ask where in the footer: `s` suggests a full
  path and `S` only a folder, since the files inside are named for you. The
  suggestions are `logs/<task>.<timestamp>.log` and `logs/<task>.<timestamp>/`,
  beside the project and named for the task you ran, so a folder of logs groups
  a task's runs together and `ls -t` still orders them by time. A `logs`
  directory that Task creates ignores itself, so it does not appear in
  `git status`; one that already exists is left alone. Any missing directories
  are created, and saved output keeps its escape sequences, so `cat` and
  `less -R` show the colour.

To take part of an output rather than all of it, select the lines you want in
the fullscreen view and press `y`. All of these work whether or not the task has
finished.

Copying uses the OSC 52 escape sequence and, where one is available, a clipboard
helper such as `wl-copy`, `pbcopy`, `xclip`, `xsel` or `clip.exe`. OSC 52 works
over SSH but is not supported everywhere; terminals based on VTE, including
GNOME Terminal, ignore it, which is why the helper is tried as well. When
neither confirmed the copy, the message says so and points at `s`.

```shell
$ task --tui --tui-task-navigator tree --tui-status labels build
```

Pressing `q` while tasks are running requests cancellation and closes the TUI
after Task's execution has returned. After execution finishes normally, the TUI
remains open so its output can be inspected; press Escape or `b` to open the
launcher, or press `q` to close it. Switching to the launcher while execution is
still in progress first cancels the tasks and waits for their processes to exit.

The TUI requires an interactive terminal. It is intended for local use; use one
of the stream-based [output modes](./output.md) in CI or when redirecting
output.

In JetBrains integrated terminals, including PyCharm, Task automatically avoids
cursor movements that JediTerm can render incorrectly. This compatibility mode
preserves terminal colours and applies only to the TUI; commands keep their
normal terminal environment.

## Prompting in the TUI {#prompting-in-the-tui}

When Task needs to ask you something, it asks in the interface. A task declaring
`prompt` shows its confirmation, and a missing required variable is asked for:
free text, or a list to choose from when the variable declares an `enum`.
`--interactive` is not needed, since the TUI can always ask; without it a task
requiring a variable could not be run from the launcher at all, as there is
nowhere to pass one.

A question appears as a dialog over the dashboard, and names the task that is
asking. A confirmation lists its answers with the default marked, so pressing
Enter gives you what you can see rather than what a `[y/N]` would have implied;
`y` and `n` still answer directly. It can arrive partway through a run, because
a task reached through `cmds` is only compiled when the run gets to it. Nothing
else can proceed until you answer.

## Limitations {#limitations}

Watch mode and tasks marked `interactive: true` are not supported. An
interactive task is not a question Task can relay: its command takes the
terminal and uses it however it likes.
