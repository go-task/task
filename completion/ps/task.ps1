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
			$listOutput | Where-Object { $_ -like "$commandName*" } | ForEach-Object { $_ }
		} else {
			$listOutput = $(task --list-all --silent)
			$reg = "^($commandName.*?)$"
			$listOutput | Where-Object { $_ -like "$commandName*" } | ForEach-Object { $_ }
	}
}
