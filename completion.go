package task

import (
	_ "embed"
	"fmt"
)

//go:embed completion/bash/task.bash
var completionBash string

//go:embed completion/fish/task.fish
var completionFish string

//go:embed completion/ps/task.ps1
var completionPowershell string

//go:embed completion/zsh/_task
var completionZsh string

// The completion/next/* scripts are thin wrappers around the `task __complete`
// engine. They are served only via `--new-completion` for now (opt-in) and will
// replace the scripts above once the engine becomes the default.

//go:embed completion/next/bash/task.bash
var completionBashNext string

//go:embed completion/next/fish/task.fish
var completionFishNext string

//go:embed completion/next/ps/task.ps1
var completionPowershellNext string

//go:embed completion/next/zsh/_task
var completionZshNext string

// Completion returns the default (stable) completion script for the given shell.
func Completion(shell string) (string, error) {
	switch shell {
	case "bash":
		return completionBash, nil
	case "fish":
		return completionFish, nil
	case "powershell":
		return completionPowershell, nil
	case "zsh":
		return completionZsh, nil
	default:
		return "", fmt.Errorf("unknown shell: %s", shell)
	}
}

// CompletionNext returns the new `task __complete` engine wrapper for the given
// shell, exposed via `--new-completion` while the engine is opt-in.
func CompletionNext(shell string) (string, error) {
	switch shell {
	case "bash":
		return completionBashNext, nil
	case "fish":
		return completionFishNext, nil
	case "powershell":
		return completionPowershellNext, nil
	case "zsh":
		return completionZshNext, nil
	default:
		return "", fmt.Errorf("unknown shell: %s", shell)
	}
}
