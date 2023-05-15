using namespace System.Management.Automation

Register-ArgumentCompleter -CommandName task -ScriptBlock {
	param($commandName, $parameterName, $wordToComplete, $commandAst, $fakeBoundParameters)
	if ($commandName.StartsWith('-')) {
		$completions = @(
			[CompletionResult]::new('--color ', '--color', [CompletionResultType]::ParameterName, '--color'),
			[CompletionResult]::new('--concurrency=', '--concurrency=', [CompletionResultType]::ParameterName, 'concurrency'),
			[CompletionResult]::new('--interval=', '--interval=', [CompletionResultType]::ParameterName, 'interval'),
			[CompletionResult]::new('--output=interleaved ', '--output=interleaved', [CompletionResultType]::ParameterName, 'output style'),
			[CompletionResult]::new('--output=group ', '--output=group', [CompletionResultType]::ParameterName, 'output style'),
			[CompletionResult]::new('--output=prefixed ', '--output=prefixed', [CompletionResultType]::ParameterName, 'output style'),
			[CompletionResult]::new('--dry ', '--dry', [CompletionResultType]::ParameterName, '--dry'),
			[CompletionResult]::new('--force ', '--force', [CompletionResultType]::ParameterName, '--force'),
			[CompletionResult]::new('--parallel ', '--parallel', [CompletionResultType]::ParameterName, '--parallel'),
			[CompletionResult]::new('--silent ', '--silent', [CompletionResultType]::ParameterName, '--silent'),
			[CompletionResult]::new('--status ', '--status', [CompletionResultType]::ParameterName, '--status'),
			[CompletionResult]::new('--verbose ', '--verbose', [CompletionResultType]::ParameterName, '--verbose'),
			[CompletionResult]::new('--watch ', '--watch', [CompletionResultType]::ParameterName, '--watch')
		)

		return $completions.Where{ $_.Tooltip -like "$commandName*" }
	}

	$tasks = $(task --list-all --json) | ConvertFrom-Json

	if ($commandName -ne '') {
		# user already input something, complete current word
		return $tasks.tasks | Where-Object { $_.name -like "$commandName*" } | ForEach-Object { $_.name }
	}

	# <tab> after <space>, show tasks with description
	return $tasks.tasks | Where-Object { $_.name -like "$commandName*" } | ForEach-Object {
		$desc = $_.name + "`n"
		if ($_.summary -ne "") {
			$desc = $_.name + "`t(" + $_.summary + ")`n"
		}
		elseif ($_.desc -ne "") {
			$desc = $_.name + "`t(" + $_.desc + ")`n"
		}

		return $desc
	}
}
