package tui

import "charm.land/bubbles/v2/key"

// mouseHelpKey marks a help entry that documents a mouse action rather than a
// key. bubbles/key only renders a binding that has at least one key, so these
// carry a sentinel no terminal can produce.
const mouseHelpKey = "\x00mouse"

// terse returns a copy of a binding with shorter help text. key.Binding is a
// value, so the original is untouched.
//
// A binding carries the description the full key list shows. The footer has
// room for a word at most, so ShortHelp restates the entries it shows. Keeping
// both wordings in one place is what stops them drifting apart.
func terse(b key.Binding, name, desc string) key.Binding {
	b.SetHelp(name, desc)
	return b
}

// fullHelpColumns lays bindings out column by column, keeping related entries
// together in reading order.
func fullHelpColumns(bindings []key.Binding, columns int) [][]key.Binding {
	columns = max(columns, 1)
	perColumn := (len(bindings) + columns - 1) / columns
	groups := make([][]key.Binding, 0, columns)
	for start := 0; start < len(bindings); start += perColumn {
		groups = append(groups, bindings[start:min(start+perColumn, len(bindings))])
	}
	return groups
}

// dashboardKeys are the two-pane view's bindings. The arrow keys mean different
// things depending on which pane has focus, so a keymap is built per render
// rather than kept as a package-level value.
type dashboardKeys struct {
	Move       key.Binding
	Pane       key.Binding
	Click      key.Binding
	Wheel      key.Binding
	Page       key.Binding
	Top        key.Binding
	Bottom     key.Binding
	Fullscreen key.Binding
	Copy       key.Binding
	CopyRaw    key.Binding
	Snapshot   key.Binding
	Launcher   key.Binding
	Quit       key.Binding
	Help       key.Binding
}

func newDashboardKeys(outputFocused, canReturnToLauncher bool) dashboardKeys {
	move := key.NewBinding(key.WithKeys("up", "down", "k", "j"), key.WithHelp("↑/↓", "select a task"))
	click := key.NewBinding(key.WithKeys(mouseHelpKey), key.WithHelp("click", "select a task"))
	if outputFocused {
		move.SetHelp("↑/↓", "scroll the output")
		click.SetHelp("click", "focus a pane")
	}
	keys := dashboardKeys{
		Move:       move,
		Pane:       key.NewBinding(key.WithKeys("tab", "shift+tab", "left", "right", "h", "l"), key.WithHelp("tab/←/→", "switch pane")),
		Click:      click,
		Wheel:      key.NewBinding(key.WithKeys(mouseHelpKey), key.WithHelp("wheel", "scroll the output")),
		Page:       key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("pgup/pgdn", "scroll a page")),
		Top:        key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("g", "jump to start")),
		Bottom:     key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("G", "jump to end")),
		Fullscreen: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "output fullscreen")),
		Copy:       key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy output without ANSI codes")),
		CopyRaw:    key.NewBinding(key.WithKeys("Y"), key.WithHelp("Y", "copy output with ANSI codes")),
		Snapshot:   key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "print output to terminal")),
		Launcher:   key.NewBinding(key.WithKeys("esc", "b"), key.WithHelp("esc/b", "stop, open launcher")),
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "stop and quit")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "show this list")),
	}
	if !canReturnToLauncher {
		keys.Launcher.SetEnabled(false)
	}
	return keys
}

// ShortHelp lists the footer keys in order of importance, because the help
// bubble drops from the end when the line does not fit. Leaving a view comes
// first: help, quit, and the way back to the launcher. Then moving around, then
// the actions.
//
// The whole line is not expected to fit eighty columns. It does not have to:
// what matters is that the entries a reader needs to get somewhere else survive,
// and "?" lists the rest.
func (k dashboardKeys) ShortHelp() []key.Binding {
	move := "select"
	if k.Move.Help().Desc == "scroll the output" {
		move = "scroll"
	}
	return []key.Binding{
		terse(k.Help, "?", "help"),
		terse(k.Quit, "q", "quit"),
		terse(k.Launcher, "esc/b", "launcher"),
		terse(k.Move, "↑/↓", move),
		terse(k.Copy, "y", "copy"),
		terse(k.Fullscreen, "f", "fullscreen"),
		terse(k.Pane, "tab", "pane"),
		terse(k.Snapshot, "t", "to terminal"),
	}
}

func (k dashboardKeys) FullHelp() [][]key.Binding {
	return fullHelpColumns(k.allBindings(), 3)
}

func (k dashboardKeys) allBindings() []key.Binding {
	return []key.Binding{
		k.Move, k.Pane, k.Click, k.Wheel, k.Page,
		k.Top, k.Bottom, k.Fullscreen, k.Copy, k.CopyRaw,
		k.Snapshot, k.Launcher, k.Quit, k.Help,
	}
}

// fullscreenKeys are the bindings of the single-pane output view.
type fullscreenKeys struct {
	Move     key.Binding
	Page     key.Binding
	Top      key.Binding
	Bottom   key.Binding
	Copy     key.Binding
	CopyRaw  key.Binding
	Snapshot key.Binding
	Return   key.Binding
	Quit     key.Binding
	Help     key.Binding
}

func newFullscreenKeys() fullscreenKeys {
	return fullscreenKeys{
		Move:     key.NewBinding(key.WithKeys("up", "down", "k", "j"), key.WithHelp("↑/↓", "scroll the output")),
		Page:     key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("pgup/pgdn", "scroll a page")),
		Top:      key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("g", "jump to start")),
		Bottom:   key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("G", "jump to end")),
		Copy:     key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy output without ANSI codes")),
		CopyRaw:  key.NewBinding(key.WithKeys("Y"), key.WithHelp("Y", "copy output with ANSI codes")),
		Snapshot: key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "print output to terminal")),
		Return:   key.NewBinding(key.WithKeys("f", "esc"), key.WithHelp("f/esc", "back to panes")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "stop and quit")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "show this list")),
	}
}

func (k fullscreenKeys) ShortHelp() []key.Binding {
	return []key.Binding{
		terse(k.Help, "?", "help"),
		terse(k.Quit, "q", "quit"),
		terse(k.Return, "f/esc", "back"),
		terse(k.Move, "↑/↓", "scroll"),
		terse(k.Copy, "y", "copy"),
		terse(k.Snapshot, "t", "to terminal"),
	}
}

func (k fullscreenKeys) FullHelp() [][]key.Binding {
	return fullHelpColumns(k.allBindings(), 3)
}

func (k fullscreenKeys) allBindings() []key.Binding {
	return []key.Binding{
		k.Move, k.Page, k.Top, k.Bottom,
		k.Copy, k.CopyRaw, k.Snapshot,
		k.Return, k.Quit, k.Help,
	}
}

// launcherKeys are the bindings of the task launcher.
//
// The launcher filters as you type, so every printable character belongs to the
// filter and cannot be a command. That rules out "?" for help here, which is why
// the launcher keeps to a single line rather than offering a full list.
type launcherKeys struct {
	Move        key.Binding
	Boundary    key.Binding
	RunInTUI    key.Binding
	RunNormally key.Binding
	ClearFilter key.Binding
	Quit        key.Binding
}

func newLauncherKeys() launcherKeys {
	return launcherKeys{
		Move:        key.NewBinding(key.WithKeys("up", "down", "tab"), key.WithHelp("↑/↓", "navigate")),
		Boundary:    key.NewBinding(key.WithKeys("home", "end"), key.WithHelp("home/end", "first/last")),
		RunInTUI:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run in TUI")),
		RunNormally: key.NewBinding(key.WithKeys("ctrl+r", "alt+enter"), key.WithHelp("ctrl+r", "run normally")),
		ClearFilter: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear")),
		Quit:        key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
	}
}

func (k launcherKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Move, k.RunInTUI, k.RunNormally, k.ClearFilter}
}

func (k launcherKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}
