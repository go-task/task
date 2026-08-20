using namespace System.Management.Automation
using namespace System.Management.Automation.Language

# Thin wrapper around `task __complete`: all suggestion logic lives in the Go engine.

$cmdNames = @('task') + (Get-Alias -Definition task,task.exe,*\task,*\task.exe -ErrorAction SilentlyContinue).Name | Select-Object -Unique

Register-ArgumentCompleter -Native -CommandName $cmdNames -ScriptBlock {
	param($wordToComplete, $commandAst, $cursorPosition)

	$TaskExe = if ($env:TASK_EXE) { $env:TASK_EXE } else { 'task' }

	# The current word arrives with the quote the user opened.
	$current = $wordToComplete
	if ($current.Length -ge 1 -and ($current[0] -eq '"' -or $current[0] -eq "'")) {
		$quoteChar = $current[0]
		$current = $current.Substring(1)
		if ($current.EndsWith($quoteChar)) {
			$current = $current.Substring(0, $current.Length - 1)
		}
	}

	# A string element yields its Value, so `--dir "a b"` arrives unquoted.
	$argsToPass = @()
	$elements = $commandAst.CommandElements
	for ($i = 1; $i -lt $elements.Count; $i++) {
		$el = $elements[$i]
		if ($el.Extent.StartOffset -ge $cursorPosition) { break }
		$argsToPass += if ($el -is [StringConstantExpressionAst] -or $el -is [ExpandableStringExpressionAst]) {
			$el.Value
		} else {
			$el.ToString()
		}
	}
	# The trailing word tells the engine the cursor is on a fresh word.
	if ($argsToPass.Count -eq 0 -or $argsToPass[-1] -ne $current) {
		$argsToPass += $current
	}

	$output = & $TaskExe __complete @argsToPass 2>$null
	if (-not $output) { return }

	$lines = @($output)
	$last = $lines[-1]
	if (-not $last.StartsWith(':')) { return }

	$directive = [int]($last.Substring(1))
	$data = if ($lines.Count -gt 1) { $lines[0..($lines.Count - 2)] } else { @() }

	# Completion directives, mirroring internal/complete/complete.go.
	$NoFileComp    = 4
	$FilterFileExt = 8
	$FilterDirs    = 16

	# PowerShell replaces the whole token, so the flag and directory prefix must
	# be prepended back to every candidate.
	$flagPrefix = ''
	$pathArg = $current
	if ($current -match '^(--?[^=]+=)(.*)$') {
		$flagPrefix = $Matches[1]
		$pathArg = $Matches[2]
	}
	$pathPrefix = $flagPrefix + ($pathArg -replace '[^\\/]*$', '')

	# DirectiveNoSpace cannot be honored: CompletionResult has no per-item "no
	# trailing space" option, so `VAR=` gets one anyway.

	# The text replaces the token as-is, so a value holding a space must be quoted.
	$asCompletionText = {
		param($text)
		if ($text -match '[\s'']') { "'" + $text.Replace("'", "''") + "'" } else { $text }
	}

	$asPathResult = {
		param($item)
		$type = if ($item.PSIsContainer) { [CompletionResultType]::ProviderContainer } else { [CompletionResultType]::ProviderItem }
		[CompletionResult]::new((& $asCompletionText "$pathPrefix$($item.Name)"), $item.Name, $type, $item.Name)
	}

	# Directories are kept so the user can descend. `-Include` needs `-Recurse`.
	if ($directive -band $FilterFileExt) {
		$exts = $data | ForEach-Object { ".$_" }
		return Get-ChildItem -Path "$pathArg*" -ErrorAction SilentlyContinue |
			Where-Object { $_.PSIsContainer -or $exts -contains $_.Extension } |
			ForEach-Object { & $asPathResult $_ }
	}

	if ($directive -band $FilterDirs) {
		return Get-ChildItem -Path "$pathArg*" -Directory -ErrorAction SilentlyContinue |
			ForEach-Object { & $asPathResult $_ }
	}

	# PowerShell does not filter native argument-completer results itself.
	$results = @($data | ForEach-Object {
		$parts = $_ -split "`t", 2
		$value = $parts[0]
		if ($current -and -not $value.StartsWith($current, [System.StringComparison]::OrdinalIgnoreCase)) { return }
		$desc = if ($parts.Count -gt 1 -and $parts[1]) { $parts[1] } else { $value }
		[CompletionResult]::new((& $asCompletionText $value), $value, [CompletionResultType]::ParameterValue, $desc)
	})

	# NoFileComp unset and nothing matched → DirectiveDefault, so offer files.
	if ($results.Count -eq 0 -and -not ($directive -band $NoFileComp)) {
		return Get-ChildItem -Path "$pathArg*" -ErrorAction SilentlyContinue |
			ForEach-Object { & $asPathResult $_ }
	}

	return $results
}
