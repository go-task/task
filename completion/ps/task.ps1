Register-ArgumentCompleter -CommandName task -ScriptBlock {
	param($commandName, $parameterName, $wordToComplete, $commandAst, $fakeBoundParameters)
		if ($commandName -match '^-{1,2}[^-]*$') {
			return @(
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
			) | Where-Object { $_ -like "$commandName*" } | ForEach-Object { $_ }
		}

		return $(task --list-all --silent) | Where-Object { $_ -like "$commandName*" } | ForEach-Object { $_ }
}
