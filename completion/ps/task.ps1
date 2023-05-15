$scriptBlock = {
	param($commandName, $parameterName, $wordToComplete, $commandAst, $fakeBoundParameters )
	$listOutput = $(task --list-all --silent)
	$listOutput | ForEach-Object { $_ }
}

Register-ArgumentCompleter -Native -CommandName task -ScriptBlock $scriptBlock
