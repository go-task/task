$scriptBlock = {
	param($commandName, $parameterName, $wordToComplete, $commandAst, $fakeBoundParameters)
		$listOutput = $(task --list-all --silent)
		$reg = "^($commandName.*?)$"
		$listOutput | Select-String $reg -AllMatches | ForEach-Object {
			 $_.Matches.Groups[1].Value;
		}
}

Register-ArgumentCompleter -CommandName task -ScriptBlock $scriptBlock
