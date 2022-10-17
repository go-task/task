$scriptBlock = {
	param($commandName, $parameterName, $wordToComplete, $commandAst, $fakeBoundParameters )
	$reg = "\* ($commandName.+?):"
	$listOutput = $(task --list-all)
	$listOutput | Select-String $reg -AllMatches | ForEach-Object { $_.Matches.Groups[1].Value }
}

Register-ArgumentCompleter -CommandName task -ScriptBlock $scriptBlock
