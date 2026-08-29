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

// The self-contained scripts that predate the engine, kept behind
// `--legacy-completion` as an escape hatch for a couple of releases.

//go:embed completion/legacy/bash/task.bash
var completionBashLegacy string

//go:embed completion/legacy/fish/task.fish
var completionFishLegacy string

//go:embed completion/legacy/nu/task-completions.nu
var completionNuLegacy string

//go:embed completion/legacy/ps/task.ps1
var completionPowershellLegacy string

//go:embed completion/legacy/zsh/_task
var completionZshLegacy string

// The maps accept `nushell` as an alias of `nu`.
var completionScripts = map[string]string{
	"bash":       completionBash,
	"fish":       completionFish,
	"nu":         completionNu,
	"nushell":    completionNu,
	"powershell": completionPowershell,
	"zsh":        completionZsh,
}

var completionScriptsLegacy = map[string]string{
	"bash":       completionBashLegacy,
	"fish":       completionFishLegacy,
	"nu":         completionNuLegacy,
	"nushell":    completionNuLegacy,
	"powershell": completionPowershellLegacy,
	"zsh":        completionZshLegacy,
}

func Completion(shell string) (string, error) {
	return completionScript(completionScripts, shell)
}

func LegacyCompletion(shell string) (string, error) {
	return completionScript(completionScriptsLegacy, shell)
}

func completionScript(scripts map[string]string, shell string) (string, error) {
	script, ok := scripts[shell]
	if !ok {
		return "", fmt.Errorf("unknown shell: %s", shell)
	}
	return script, nil
}
