package log

import (
	"io"
	"os"
	"strconv"

	"github.com/fatih/color"
)

type (
	Color     func() PrintFunc
	PrintFunc func(io.Writer, string, ...any)
)

func Default() PrintFunc {
	return color.New(envColor("TASK_COLOR_RESET", color.Reset)).FprintfFunc()
}

func Blue() PrintFunc {
	return color.New(envColor("TASK_COLOR_BLUE", color.FgBlue)).FprintfFunc()
}

func Green() PrintFunc {
	return color.New(envColor("TASK_COLOR_GREEN", color.FgGreen)).FprintfFunc()
}

func Cyan() PrintFunc {
	return color.New(envColor("TASK_COLOR_CYAN", color.FgCyan)).FprintfFunc()
}

func Yellow() PrintFunc {
	return color.New(envColor("TASK_COLOR_YELLOW", color.FgYellow)).FprintfFunc()
}

func Magenta() PrintFunc {
	return color.New(envColor("TASK_COLOR_MAGENTA", color.FgMagenta)).FprintfFunc()
}

func Red() PrintFunc {
	return color.New(envColor("TASK_COLOR_RED", color.FgRed)).FprintfFunc()
}

func envColor(env string, defaultColor color.Attribute) color.Attribute {
	if os.Getenv("FORCE_COLOR") != "" {
		color.NoColor = false
	}

	override, err := strconv.Atoi(os.Getenv(env))
	if err == nil {
		return color.Attribute(override)
	}
	return defaultColor
}
