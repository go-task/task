using namespace System.Management.Automation

# Thin wrapper around `task __complete`. All suggestion logic lives in the
# Go engine — do not add completion logic here.

$cmdNames = @('task') + (Get-Alias -Definition task,task.exe,*\task,*\task.exe -ErrorAction SilentlyContinue).Name | Select-Object -Unique

Register-ArgumentCompleter -Native -CommandName $cmdNames -ScriptBlock {
	param($wordToComplete, $commandAst, $cursorPosition)

	$TaskExe = if ($env:TASK_EXE) { $env:TASK_EXE } else { 'task' }

	# Words after the program name, truncated to the cursor.
	$argsToPass = @()
	$elements = $commandAst.CommandElements
	if ($elements.Count -gt 1) {
		for ($i = 1; $i -lt $elements.Count; $i++) {
			$el = $elements[$i]
			if ($el.Extent.StartOffset -ge $cursorPosition) { break }
			$argsToPass += $el.ToString()
		}
	}
	# The trailing word (possibly empty) must reach the engine so it knows
	# the cursor sits on a fresh word. It is already present when it coincides
	# with the last command element captured above.
	if ($argsToPass.Count -eq 0 -or $argsToPass[-1] -ne $wordToComplete) {
		$argsToPass += $wordToComplete
	}

	$output = & $TaskExe __complete @argsToPass 2>$null
	if (-not $output) { return }

	$lines = @($output)
	if ($lines.Count -eq 0) { return }
	$last = $lines[-1]
	if (-not $last.StartsWith(':')) { return }

	$directive = [int]($last.Substring(1))
	$data = if ($lines.Count -gt 1) { $lines[0..($lines.Count - 2)] } else { @() }

	# Note: DirectiveNoSpace (bit 2) cannot be honored here — PowerShell's
	# CompletionResult API has no per-item "no trailing space" option, so a
	# suggestion like `VAR=` gets a trailing space. This is a PowerShell limit.

	# FilterFileExt
	if ($directive -band 8) {
		$patterns = $data | ForEach-Object { "*.$_" }
		return Get-ChildItem -Path . -Include $patterns -File -ErrorAction SilentlyContinue |
			ForEach-Object { [CompletionResult]::new($_.Name, $_.Name, [CompletionResultType]::ProviderItem, $_.Name) }
	}

	# FilterDirs
	if ($directive -band 16) {
		return Get-ChildItem -Path . -Directory -ErrorAction SilentlyContinue |
			ForEach-Object { [CompletionResult]::new($_.Name, $_.Name, [CompletionResultType]::ProviderContainer, $_.Name) }
	}

	# Build candidates, filtering by the current word. PowerShell does not filter
	# native argument-completer results itself, so without this every suggestion
	# would be offered regardless of what the user typed.
	$results = @($data | ForEach-Object {
		$parts = $_ -split "`t", 2
		$value = $parts[0]
		if ($wordToComplete -and -not $value.StartsWith($wordToComplete)) { return }
		$desc = if ($parts.Count -gt 1 -and $parts[1]) { $parts[1] } else { $value }
		[CompletionResult]::new($value, $value, [CompletionResultType]::ParameterValue, $desc)
	})

	# NoFileComp (bit 4) unset and nothing matched → fall back to file completion,
	# since the engine returned DirectiveDefault (e.g. --cacert, after `--`).
	if ($results.Count -eq 0 -and -not ($directive -band 4)) {
		return Get-ChildItem -Path . -ErrorAction SilentlyContinue |
			ForEach-Object { [CompletionResult]::new($_.Name, $_.Name, [CompletionResultType]::ProviderItem, $_.Name) }
	}

	return $results
}
