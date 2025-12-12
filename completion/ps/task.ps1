using namespace System.Management.Automation

Register-ArgumentCompleter -CommandName task -ScriptBlock {
	param($commandName, $parameterName, $wordToComplete, $commandAst, $fakeBoundParameters)

	if ($commandName.StartsWith('-')) {
		$completions = @(
			# Standard flags (alphabetical order)
			[CompletionResult]::new('-a', '-a', [CompletionResultType]::ParameterName, 'list all tasks'),
			[CompletionResult]::new('--list-all', '--list-all', [CompletionResultType]::ParameterName, 'list all tasks'),
			[CompletionResult]::new('-c', '-c', [CompletionResultType]::ParameterName, 'colored output'),
			[CompletionResult]::new('--color', '--color', [CompletionResultType]::ParameterName, 'colored output'),
			[CompletionResult]::new('-C', '-C', [CompletionResultType]::ParameterName, 'limit concurrent tasks'),
			[CompletionResult]::new('--concurrency', '--concurrency', [CompletionResultType]::ParameterName, 'limit concurrent tasks'),
			[CompletionResult]::new('--completion', '--completion', [CompletionResultType]::ParameterName, 'generate shell completion'),
			[CompletionResult]::new('-d', '-d', [CompletionResultType]::ParameterName, 'set directory'),
			[CompletionResult]::new('--dir', '--dir', [CompletionResultType]::ParameterName, 'set directory'),
			[CompletionResult]::new('--disable-fuzzy', '--disable-fuzzy', [CompletionResultType]::ParameterName, 'disable fuzzy matching'),
			[CompletionResult]::new('-n', '-n', [CompletionResultType]::ParameterName, 'dry run'),
			[CompletionResult]::new('--dry', '--dry', [CompletionResultType]::ParameterName, 'dry run'),
			[CompletionResult]::new('-x', '-x', [CompletionResultType]::ParameterName, 'pass-through exit code'),
			[CompletionResult]::new('--exit-code', '--exit-code', [CompletionResultType]::ParameterName, 'pass-through exit code'),
			[CompletionResult]::new('--experiments', '--experiments', [CompletionResultType]::ParameterName, 'list experiments'),
			[CompletionResult]::new('-F', '-F', [CompletionResultType]::ParameterName, 'fail fast on pallalel tasks'),
			[CompletionResult]::new('--failfast', '--failfast', [CompletionResultType]::ParameterName, 'force execution'),
			[CompletionResult]::new('-f', '-f', [CompletionResultType]::ParameterName, 'force execution'),
			[CompletionResult]::new('--force', '--force', [CompletionResultType]::ParameterName, 'force execution'),
			[CompletionResult]::new('-g', '-g', [CompletionResultType]::ParameterName, 'run global Taskfile'),
			[CompletionResult]::new('--global', '--global', [CompletionResultType]::ParameterName, 'run global Taskfile'),
			[CompletionResult]::new('-h', '-h', [CompletionResultType]::ParameterName, 'show help'),
			[CompletionResult]::new('--help', '--help', [CompletionResultType]::ParameterName, 'show help'),
			[CompletionResult]::new('-i', '-i', [CompletionResultType]::ParameterName, 'create new Taskfile'),
			[CompletionResult]::new('--init', '--init', [CompletionResultType]::ParameterName, 'create new Taskfile'),
			[CompletionResult]::new('--insecure', '--insecure', [CompletionResultType]::ParameterName, 'allow insecure downloads'),
			[CompletionResult]::new('-I', '-I', [CompletionResultType]::ParameterName, 'watch interval'),
			[CompletionResult]::new('--interval', '--interval', [CompletionResultType]::ParameterName, 'watch interval'),
			[CompletionResult]::new('-j', '-j', [CompletionResultType]::ParameterName, 'format as JSON'),
			[CompletionResult]::new('--json', '--json', [CompletionResultType]::ParameterName, 'format as JSON'),
			[CompletionResult]::new('-l', '-l', [CompletionResultType]::ParameterName, 'list tasks'),
			[CompletionResult]::new('--list', '--list', [CompletionResultType]::ParameterName, 'list tasks'),
			[CompletionResult]::new('--nested', '--nested', [CompletionResultType]::ParameterName, 'nest namespaces in JSON'),
			[CompletionResult]::new('--no-status', '--no-status', [CompletionResultType]::ParameterName, 'ignore status in JSON'),
			[CompletionResult]::new('-o', '-o', [CompletionResultType]::ParameterName, 'set output style'),
			[CompletionResult]::new('--output', '--output', [CompletionResultType]::ParameterName, 'set output style'),
			[CompletionResult]::new('--output-group-begin', '--output-group-begin', [CompletionResultType]::ParameterName, 'template before group'),
			[CompletionResult]::new('--output-group-end', '--output-group-end', [CompletionResultType]::ParameterName, 'template after group'),
			[CompletionResult]::new('--output-group-error-only', '--output-group-error-only', [CompletionResultType]::ParameterName, 'hide successful output'),
			[CompletionResult]::new('-p', '-p', [CompletionResultType]::ParameterName, 'execute in parallel'),
			[CompletionResult]::new('--parallel', '--parallel', [CompletionResultType]::ParameterName, 'execute in parallel'),
			[CompletionResult]::new('-s', '-s', [CompletionResultType]::ParameterName, 'silent mode'),
			[CompletionResult]::new('--silent', '--silent', [CompletionResultType]::ParameterName, 'silent mode'),
			[CompletionResult]::new('--sort', '--sort', [CompletionResultType]::ParameterName, 'task sorting order'),
			[CompletionResult]::new('--status', '--status', [CompletionResultType]::ParameterName, 'check task status'),
			[CompletionResult]::new('--summary', '--summary', [CompletionResultType]::ParameterName, 'show task summary'),
			[CompletionResult]::new('-t', '-t', [CompletionResultType]::ParameterName, 'choose Taskfile'),
			[CompletionResult]::new('--taskfile', '--taskfile', [CompletionResultType]::ParameterName, 'choose Taskfile'),
			[CompletionResult]::new('-v', '-v', [CompletionResultType]::ParameterName, 'verbose output'),
			[CompletionResult]::new('--verbose', '--verbose', [CompletionResultType]::ParameterName, 'verbose output'),
			[CompletionResult]::new('--version', '--version', [CompletionResultType]::ParameterName, 'show version'),
			[CompletionResult]::new('-w', '-w', [CompletionResultType]::ParameterName, 'watch mode'),
			[CompletionResult]::new('--watch', '--watch', [CompletionResultType]::ParameterName, 'watch mode'),
			[CompletionResult]::new('-y', '-y', [CompletionResultType]::ParameterName, 'assume yes'),
			[CompletionResult]::new('--yes', '--yes', [CompletionResultType]::ParameterName, 'assume yes')
		)

		# Experimental flags (dynamically added based on enabled experiments)
		$experiments = & task --experiments 2>$null | Out-String

		if ($experiments -match '\* GENTLE_FORCE:.*on') {
			$completions += [CompletionResult]::new('--force-all', '--force-all', [CompletionResultType]::ParameterName, 'force all dependencies')
		}

		if ($experiments -match '\* REMOTE_TASKFILES:.*on') {
			# Options
			$completions += [CompletionResult]::new('--offline', '--offline', [CompletionResultType]::ParameterName, 'use cached Taskfiles')
			$completions += [CompletionResult]::new('--timeout', '--timeout', [CompletionResultType]::ParameterName, 'download timeout')
			$completions += [CompletionResult]::new('--expiry', '--expiry', [CompletionResultType]::ParameterName, 'cache expiry')
			$completions += [CompletionResult]::new('--remote-cache-dir', '--remote-cache-dir', [CompletionResultType]::ParameterName, 'cache directory')
			# Operations
			$completions += [CompletionResult]::new('--download', '--download', [CompletionResultType]::ParameterName, 'download remote Taskfile')
			$completions += [CompletionResult]::new('--clear-cache', '--clear-cache', [CompletionResultType]::ParameterName, 'clear cache')
		}

		return $completions.Where{ $_.CompletionText.StartsWith($commandName) }
	}

	return 	$(task --list-all --silent) | Where-Object { $_.StartsWith($commandName) } | ForEach-Object { return $_ + " " }
}
