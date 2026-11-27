package execext

import (
	"fmt"
	"strconv"

	"charm.land/turtle/shcoreutils"
	"mvdan.cc/sh/v3/interp"

	"github.com/go-task/task/v3/internal/env"
)

// CoreutilsMode selects how the core utilities run in Task's shell.
type CoreutilsMode string

const (
	// CoreutilsTurtle always uses the charm.land/turtle implementations.
	CoreutilsTurtle CoreutilsMode = "turtle"
	// CoreutilsSystem prefers the system utilities, falling back to Turtle
	// when one is not found in PATH.
	CoreutilsSystem CoreutilsMode = "system"
	// CoreutilsAuto uses Turtle on Windows and the system utilities
	// elsewhere.
	CoreutilsAuto CoreutilsMode = "auto"
	// CoreutilsOff skips the coreutils middleware entirely.
	CoreutilsOff CoreutilsMode = "off"
)

// DefaultCoreutils is the mode used when neither the Taskfile nor the
// environment selects one.
const DefaultCoreutils = CoreutilsAuto

// ParseCoreutilsMode returns the mode named by value, or an error naming the
// valid ones. An empty value means the default.
func ParseCoreutilsMode(value string) (CoreutilsMode, error) {
	switch mode := CoreutilsMode(value); mode {
	case "", CoreutilsTurtle, CoreutilsSystem, CoreutilsAuto, CoreutilsOff:
		if mode == "" {
			return DefaultCoreutils, nil
		}
		return mode, nil
	default:
		return "", fmt.Errorf("invalid coreutils mode %q, expected turtle, system, auto, or off", value)
	}
}

// CoreutilsFromEnv returns the mode set in the environment, reporting whether
// one was selected. TASK_COREUTILS is checked first, with the legacy
// TASK_CORE_UTILS as a fallback. Both accept a mode name or, for backward
// compatibility, a boolean: true maps to system and false to auto. Values
// that are neither are ignored so the remaining fallbacks apply.
func CoreutilsFromEnv() (CoreutilsMode, bool) {
	for _, key := range []string{"COREUTILS", "CORE_UTILS"} {
		value := env.GetTaskEnv(key)
		if value == "" {
			continue
		}
		if mode, err := ParseCoreutilsMode(value); err == nil {
			return mode, true
		}
		if b, err := strconv.ParseBool(value); err == nil {
			if b {
				return CoreutilsSystem, true
			}
			return CoreutilsAuto, true
		}
	}
	return DefaultCoreutils, false
}

// ResolveCoreutils returns the mode set in the environment, falling back to
// the taskrc value and then the default.
func ResolveCoreutils(taskrcValue string) (CoreutilsMode, error) {
	if mode, set := CoreutilsFromEnv(); set {
		return mode, nil
	}
	if taskrcValue != "" {
		return ParseCoreutilsMode(taskrcValue)
	}
	return DefaultCoreutils, nil
}

// coreutilsHandler builds the Turtle coreutils middleware for mode, or nil
// when mode is off.
func coreutilsHandler(mode CoreutilsMode) func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	switch mode {
	case CoreutilsOff:
		return nil
	case CoreutilsSystem:
		return shcoreutils.New(shcoreutils.WithPriority(shcoreutils.PrioritySystem))
	case CoreutilsAuto:
		return shcoreutils.New(shcoreutils.WithPriority(shcoreutils.PriorityAuto))
	default:
		return shcoreutils.New(shcoreutils.WithPriority(shcoreutils.PriorityTurtle))
	}
}
