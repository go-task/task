package logger

import (
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/fatih/color"

	"github.com/go-task/task/v3/internal/env"
)

var (
	attrsReset       = envColor("COLOR_RESET", color.Reset)
	attrsFgBlue      = envColor("COLOR_BLUE", color.FgBlue)
	attrsFgGreen     = envColor("COLOR_GREEN", color.FgGreen)
	attrsFgCyan      = envColor("COLOR_CYAN", color.FgCyan)
	attrsFgYellow    = envColor("COLOR_YELLOW", color.FgYellow)
	attrsFgMagenta   = envColor("COLOR_MAGENTA", color.FgMagenta)
	attrsFgRed       = envColor("COLOR_RED", color.FgRed)
	attrsFgHiBlue    = envColor("COLOR_BRIGHT_BLUE", color.FgHiBlue)
	attrsFgHiGreen   = envColor("COLOR_BRIGHT_GREEN", color.FgHiGreen)
	attrsFgHiCyan    = envColor("COLOR_BRIGHT_CYAN", color.FgHiCyan)
	attrsFgHiYellow  = envColor("COLOR_BRIGHT_YELLOW", color.FgHiYellow)
	attrsFgHiMagenta = envColor("COLOR_BRIGHT_MAGENTA", color.FgHiMagenta)
	attrsFgHiRed     = envColor("COLOR_BRIGHT_RED", color.FgHiRed)
)

type (
	Color     func() PrintFunc
	PrintFunc func(io.Writer, string, ...any)
)

func Default() PrintFunc {
	return color.New(attrsReset...).FprintfFunc()
}

func Blue() PrintFunc {
	return color.New(attrsFgBlue...).FprintfFunc()
}

func Green() PrintFunc {
	return color.New(attrsFgGreen...).FprintfFunc()
}

func Cyan() PrintFunc {
	return color.New(attrsFgCyan...).FprintfFunc()
}

func Yellow() PrintFunc {
	return color.New(attrsFgYellow...).FprintfFunc()
}

func Magenta() PrintFunc {
	return color.New(attrsFgMagenta...).FprintfFunc()
}

func Red() PrintFunc {
	return color.New(attrsFgRed...).FprintfFunc()
}

func BrightBlue() PrintFunc {
	return color.New(attrsFgHiBlue...).FprintfFunc()
}

func BrightGreen() PrintFunc {
	return color.New(attrsFgHiGreen...).FprintfFunc()
}

func BrightCyan() PrintFunc {
	return color.New(attrsFgHiCyan...).FprintfFunc()
}

func BrightYellow() PrintFunc {
	return color.New(attrsFgHiYellow...).FprintfFunc()
}

func BrightMagenta() PrintFunc {
	return color.New(attrsFgHiMagenta...).FprintfFunc()
}

func BrightRed() PrintFunc {
	return color.New(attrsFgHiRed...).FprintfFunc()
}

func envColor(name string, defaultColor color.Attribute) []color.Attribute {
	// Fetch the environment variable
	override := env.GetTaskEnv(name)

	// First, try splitting the string by commas (RGB shortcut syntax) and if it
	// matches, then prepend the 256-color foreground escape sequence.
	// Otherwise, split by semicolons (ANSI color codes) and use them as is.
	attributeStrs := strings.Split(override, ",")
	if len(attributeStrs) == 3 {
		attributeStrs = slices.Concat([]string{"38", "2"}, attributeStrs)
	} else {
		attributeStrs = strings.Split(override, ";")
	}

	// Loop over the attributes and convert them to integers
	attributes := make([]color.Attribute, len(attributeStrs))
	for i, attributeStr := range attributeStrs {
		attribute, err := strconv.Atoi(attributeStr)
		if err != nil {
			return []color.Attribute{defaultColor}
		}
		attributes[i] = color.Attribute(attribute)
	}

	return attributes
}
