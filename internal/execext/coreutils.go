package execext

import (
	"runtime"
	"strconv"

	"github.com/go-task/task/v3/internal/env"
)

var useGoCoreUtils bool

func init() {
	// If TASK_CORE_UTILS is set to either true or false, respect that.
	// By default, enable on Windows only.
	if v, err := strconv.ParseBool(env.GetTaskEnv("CORE_UTILS")); err == nil {
		useGoCoreUtils = v
	} else {
		useGoCoreUtils = runtime.GOOS == "windows"
	}
}
