using namespace System.Management.Automation

Register-ArgumentCompleter -CommandName task -ScriptBlock {
	param($commandName, $parameterName, $wordToComplete, $commandAst, $fakeBoundParameters)

	if ($commandName.StartsWith('-')) {
		$completions = @(
			[CompletionResult]::new('--list-all ', '--list-all ', [CompletionResultType]::ParameterName, 'list all tasks'),
			[CompletionResult]::new('--color ', '--color', [CompletionResultType]::ParameterName, '--color'),
			[CompletionResult]::new('--concurrency=', '--concurrency=', [CompletionResultType]::ParameterName, 'concurrency'),
			[CompletionResult]::new('--interval=', '--interval=', [CompletionResultType]::ParameterName, 'interval'),
			[CompletionResult]::new('--output=interleaved ', '--output=interleaved', [CompletionResultType]::ParameterName, '--output='),
			[CompletionResult]::new('--output=group ', '--output=group', [CompletionResultType]::ParameterName, '--output='),
			[CompletionResult]::new('--output=prefixed ', '--output=prefixed', [CompletionResultType]::ParameterName, '--output='),
			[CompletionResult]::new('--dry ', '--dry', [CompletionResultType]::ParameterName, '--dry'),
			[CompletionResult]::new('--force ', '--force', [CompletionResultType]::ParameterName, '--force'),
			[CompletionResult]::new('--parallel ', '--parallel', [CompletionResultType]::ParameterName, '--parallel'),
			[CompletionResult]::new('--silent ', '--silent', [CompletionResultType]::ParameterName, '--silent'),
			[CompletionResult]::new('--status ', '--status', [CompletionResultType]::ParameterName, '--status'),
			[CompletionResult]::new('--verbose ', '--verbose', [CompletionResultType]::ParameterName, '--verbose'),
			[CompletionResult]::new('--watch ', '--watch', [CompletionResultType]::ParameterName, '--watch')
		)

		return $completions.Where{ $_.CompletionText.StartsWith($commandName) }
	}

	return 	$(task --list-all --silent) | Where-Object { $_.StartsWith($commandName) } | ForEach-Object { return $_ + " " }
}
