package task

import (
	_ "embed"
	"fmt"
)

// Thin wrappers around the `task __complete` engine, served by `--completion`.

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

// Accept `nushell` as an alias of `nu`.
var completionScripts = map[string]string{
	"bash":       completionBash,
	"fish":       completionFish,
	"nu":         completionNu,
	"nushell":    completionNu,
	"powershell": completionPowershell,
	"zsh":        completionZsh,
}

func Completion(shell string) (string, error) {
	script, ok := completionScripts[shell]
	if !ok {
		return "", fmt.Errorf("unknown shell: %s", shell)
	}
	return script, nil
}
