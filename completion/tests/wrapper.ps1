# Tests the PowerShell wrapper end-to-end via the completion API, which returns
# the real completions of a command line without a TTY. The `task` command must
# resolve to the binary under test (run.sh puts a symlink on PATH).
#
# Requires: $env:TASK_FIXTURE (dir with a Taskfile.yml and sample files/dirs).

Set-Location $env:TASK_FIXTURE
. "$PSScriptRoot/../ps/task.ps1"

$fails = 0

function Cands($line) {
	([System.Management.Automation.CommandCompletion]::CompleteInput($line, $line.Length, $null)).CompletionMatches |
		ForEach-Object { $_.CompletionText }
}

function Has($label, $line, $value) {
	if ((Cands $line) -contains $value) {
		Write-Output "  ok   $label"
	} else {
		Write-Output "  FAIL $label — '$value' missing from: $((Cands $line) -join ' ')"
		$script:fails++
	}
}

function HasNot($label, $line, $value) {
	if ((Cands $line) -contains $value) {
		Write-Output "  FAIL $label — '$value' should be absent"
		$script:fails++
	} else {
		Write-Output "  ok   $label"
	}
}

Write-Output "powershell: task names (no files)"
Has    "lists tasks"        'task ' 'build'
Has    "lists aliases"      'task ' 'dep'
HasNot "no files for tasks" 'task ' 'notes.txt'

Write-Output "powershell: prefix filtering"
Has    "filters by prefix"  'task b' 'build'
HasNot "prefix excludes"    'task b' 'deploy'

Write-Output "powershell: task variables"
Has    "required vars"      'task deploy ' 'ENV=dev'

Write-Output "powershell: flag values"
Has    "enum values"        'task --output ' 'interleaved'

Write-Output "powershell: --dir completes directories only"
Has    "dirs offered"       'task --dir ' 'sub'
HasNot "no plain files"     'task --dir ' 'notes.txt'

Write-Output "powershell: --taskfile filters by extension"
Has    "yaml offered"       'task --taskfile ' 'Taskfile.yml'
HasNot "txt filtered out"   'task --taskfile ' 'notes.txt'

if ($fails -ne 0) {
	Write-Output "powershell: $fails failure(s)"
	exit 1
}
Write-Output "powershell: all passed"
