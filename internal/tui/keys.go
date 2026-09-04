package tui

import "charm.land/bubbles/v2/key"

// mouseHelpKey marks a help entry that documents a mouse action rather than a
// key. bubbles/key only renders a binding that has at least one key, so these
// carry a sentinel no terminal can produce.
const mouseHelpKey = "\x00mouse"

// terse returns a copy of a binding with shorter help text, for the one-line
// footer. key.Binding is a value, so the original is untouched.
func terse(b key.Binding, name, desc string) key.Binding {
	b.SetHelp(name, desc)
	return b
}

// dashboardKeys are the two-pane view's bindings. Help text for the arrow keys
// depends on which pane has focus, so a keymap is built per render rather than
// kept as a package-level value.
type dashboardKeys struct {
	Pane       key.Binding
	Move       key.Binding
	Page       key.Binding
	Top        key.Binding
	Bottom     key.Binding
	Click      key.Binding
	Wheel      key.Binding
	Fullscreen key.Binding
	Copy       key.Binding
	CopyRaw    key.Binding
	Snapshot   key.Binding
	Launcher   key.Binding
	Quit       key.Binding
	Help       key.Binding
}

func newDashboardKeys(outputFocused, canReturnToLauncher bool) dashboardKeys {
	move := key.NewBinding(key.WithKeys("up", "down", "k", "j"), key.WithHelp("↑/↓", "select"))
	click := key.NewBinding(key.WithKeys(mouseHelpKey), key.WithHelp("click", "select"))
	if outputFocused {
		move.SetHelp("↑/↓", "scroll")
		click.SetHelp("click", "focus pane")
	}
	keys := dashboardKeys{
		Pane:       key.NewBinding(key.WithKeys("tab", "shift+tab", "left", "right", "h", "l"), key.WithHelp("tab/←/→", "pane")),
		Move:       move,
		Page:       key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("pgup/pgdn", "page")),
		Top:        key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("g", "top")),
		Bottom:     key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("G", "bottom")),
		Click:      click,
		Wheel:      key.NewBinding(key.WithKeys(mouseHelpKey), key.WithHelp("wheel", "scroll")),
		Fullscreen: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "fullscreen")),
		Copy:       key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy")),
		CopyRaw:    key.NewBinding(key.WithKeys("Y"), key.WithHelp("Y", "copy with colours")),
		Snapshot:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "snapshot to terminal")),
		Launcher:   key.NewBinding(key.WithKeys("esc", "b"), key.WithHelp("esc/b", "launcher")),
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "keys")),
	}
	if !canReturnToLauncher {
		keys.Launcher.SetEnabled(false)
	}
	return keys
}

// ShortHelp lists the footer keys in the order they matter, because the help
// bubble drops from the end when the line does not fit.
//
// Descriptions are terser than in the full list so that the whole line, quit
// and the pointer to the full list included, survives an 80 column terminal.
func (k dashboardKeys) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Move,
		terse(k.Pane, "tab", "pane"),
		k.Fullscreen,
		k.Copy,
		terse(k.Snapshot, "s", "snapshot"),
		k.Quit,
		k.Help,
	}
}

func (k dashboardKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Move, k.Pane, k.Click, k.Wheel},
		{k.Page, k.Top, k.Bottom, k.Fullscreen},
		{k.Copy, k.CopyRaw, k.Snapshot},
		{k.Launcher, k.Quit, k.Help},
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
		Move:     key.NewBinding(key.WithKeys("up", "down", "k", "j"), key.WithHelp("↑/↓", "scroll")),
		Page:     key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("pgup/pgdn", "page")),
		Top:      key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("g", "top")),
		Bottom:   key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("G", "bottom")),
		Copy:     key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy")),
		CopyRaw:  key.NewBinding(key.WithKeys("Y"), key.WithHelp("Y", "copy with colours")),
		Snapshot: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "snapshot to terminal")),
		Return:   key.NewBinding(key.WithKeys("f", "esc"), key.WithHelp("f/esc", "return")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "keys")),
	}
}

func (k fullscreenKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Move, k.Copy, terse(k.Snapshot, "s", "snapshot"), k.Return, k.Help}
}

func (k fullscreenKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Move, k.Page, k.Top, k.Bottom},
		{k.Copy, k.CopyRaw, k.Snapshot},
		{k.Return, k.Quit, k.Help},
	}
}

// launcherKeys are the bindings of the task launcher.
//
// The launcher filters as you type, so every printable character belongs to the
// filter and cannot be a command. That rules out "?" for help here, which is why
// the launcher keeps to a single line of help rather than offering a full view.
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
	return []key.Binding{k.Move, k.RunInTUI, k.RunNormally, k.ClearFilter, k.Quit}
}

func (k launcherKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}
