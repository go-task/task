# Smoke-tests how the PowerShell wrapper INTERPRETS each directive (files vs
# dirs vs no files) plus its own prefix filtering, via the completion API which
# returns real completions without a TTY. The suggestion logic (which
# tasks/aliases/vars) is covered by the Go tests; here we only check routing.
# `task` must resolve to the binary under test (run.sh puts a symlink on PATH).
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

Write-Output "powershell: :4 (NoFileComp) forwards candidates, offers no files"
Has    "candidate forwarded" 'task ' 'build'
HasNot "no file fallback"    'task ' 'notes.txt'

Write-Output "powershell: filters candidates by the current word"
Has    "prefix keeps match"  'task b' 'build'
HasNot "prefix drops others" 'task b' 'deploy'

Write-Output "powershell: :16 (FilterDirs) offers directories only"
Has    "dir offered"         'task --dir ' 'sub'
HasNot "no plain file"       'task --dir ' 'notes.txt'

Write-Output "powershell: :8 (FilterFileExt) filters by extension"
Has    "matching file"       'task --taskfile ' 'Taskfile.yml'
HasNot "non-matching file"   'task --taskfile ' 'notes.txt'

if ($fails -ne 0) {
	Write-Output "powershell: $fails failure(s)"
	exit 1
}
Write-Output "powershell: all passed"
