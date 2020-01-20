$scriptBlock = {
	param($commandName, $wordToComplete, $cursorPosition)
	$curReg = "task{.exe}? (.*?)$"
	$startsWith = $wordToComplete | Select-String $curReg -AllMatches | ForEach-Object { $_.Matches.Groups[1].Value }
	$reg = "\* ($startsWith.+?):"
	$listOutput = $(task -l)
	$listOutput | Select-String $reg -AllMatches | ForEach-Object { $_.Matches.Groups[1].Value + " " }
}

Register-ArgumentCompleter -Native -CommandName task -ScriptBlock $scriptBlock
