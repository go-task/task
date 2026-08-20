# Smoke-tests how the PowerShell wrapper routes each directive, via the
# completion API. Set up by run.sh: $env:TASK_FIXTURE, and `task` on PATH =
# the binary under test.

Set-Location $env:TASK_FIXTURE
. "$PSScriptRoot/../next/ps/task.ps1"

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

Write-Output "powershell: nested path completion keeps the directory prefix"
Has    "prefix kept"         'task --taskfile sub/' 'sub/nested.yml'

Write-Output "powershell: inline --flag=path keeps the --flag= prefix"
Has    "inline nested"       'task --taskfile=sub/' '--taskfile=sub/nested.yml'
HasNot "inline non-matching" 'task --taskfile=' '--taskfile=notes.txt'

Write-Output "powershell: a quoted argument reaches the engine unquoted"
Has    "single-quoted dir"   "task --dir 'with space' " 'spaced'
Has    "double-quoted dir"   'task --dir "with space" ' 'spaced'

Write-Output "powershell: a candidate holding a space is quoted for insertion"
Has    "dir quoted"          'task --dir w' "'with space'"

if ($fails -ne 0) {
	Write-Output "powershell: $fails failure(s)"
	exit 1
}
Write-Output "powershell: all passed"
