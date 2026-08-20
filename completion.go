package task

import (
	_ "embed"
	"fmt"
)

//go:embed completion/bash/task.bash
var completionBash string

//go:embed completion/fish/task.fish
var completionFish string

//go:embed completion/nu/task-completions.nu
var completionNu string

//go:embed completion/ps/task.ps1
var completionPowershell string

//go:embed completion/zsh/_task
var completionZsh string

// Thin wrappers around the `task __complete` engine, served via
// `--new-completion` until the engine becomes the default.

//go:embed completion/next/bash/task.bash
var completionBashNext string

//go:embed completion/next/fish/task.fish
var completionFishNext string

//go:embed completion/next/nu/task-completions.nu
var completionNuNext string

//go:embed completion/next/ps/task.ps1
var completionPowershellNext string

//go:embed completion/next/zsh/_task
var completionZshNext string

// The maps accept `nushell` as an alias of `nu`.
var completionScripts = map[string]string{
	"bash":       completionBash,
	"fish":       completionFish,
	"nu":         completionNu,
	"nushell":    completionNu,
	"powershell": completionPowershell,
	"zsh":        completionZsh,
}

var completionScriptsNext = map[string]string{
	"bash":       completionBashNext,
	"fish":       completionFishNext,
	"nu":         completionNuNext,
	"nushell":    completionNuNext,
	"powershell": completionPowershellNext,
	"zsh":        completionZshNext,
}

func Completion(shell string) (string, error) {
	return completionScript(completionScripts, shell)
}

func CompletionNext(shell string) (string, error) {
	return completionScript(completionScriptsNext, shell)
}

func completionScript(scripts map[string]string, shell string) (string, error) {
	script, ok := scripts[shell]
	if !ok {
		return "", fmt.Errorf("unknown shell: %s", shell)
	}
	return script, nil
}
