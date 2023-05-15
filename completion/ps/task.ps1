Register-ArgumentCompleter -CommandName task -ScriptBlock {
	param($commandName, $parameterName, $wordToComplete, $commandAst, $fakeBoundParameters)
		if ($commandName -match '^-{1,2}[^-]*$') {
			$listOutput = @(
				'--concurrency=',
				'--interval=',
				'--output=interleaved',
				'--output=group',
				'--output=prefixed',
				'--color',
				'--dry',
				'--force',
				'--parallel',
				'--silent',
				'--status',
				'--verbose',
				'--watch'
			)
			$reg = "^($commandName.*?)$"
			$listOutput | Select-String $reg -AllMatches | ForEach-Object {
				$_.Matches.Groups[1].Value;
			}
		} else {
			$listOutput = $(task --list-all --silent)
			$reg = "^($commandName.*?)$"
			$listOutput | Select-String $reg -AllMatches | ForEach-Object {
				$_.Matches.Groups[1].Value;
			}
	}
}
