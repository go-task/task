# Changelog

## v3.2.2

- Improve performance of `--list` and `--summary` by skipping running shell
  variables for these flags
  ([#332](https://github.com/go-task/task/issues/332)).
- Fixed a bug where an environment in a Taskfile was not always overridable
  by the system environment
  ([#425](https://github.com/go-task/task/issues/425)).
- Fixed environment from .env files not being available as variables
  ([#379](https://github.com/go-task/task/issues/379)).
- The install script is now working for ARM platforms
  ([#428](https://github.com/go-task/task/pull/428)).

## v3.2.1

- Fixed some bugs and regressions regarding dynamic variables and directories
  ([#426](https://github.com/go-task/task/issues/426)).
- The [slim-sprig](https://github.com/go-task/slim-sprig) package was updated
  with the upstream [sprig](https://github.com/Masterminds/sprig).

## v3.2.0

- Fix the `.task` directory being created in the task directory instead of the
  Taskfile directory
  ([#247](https://github.com/go-task/task/issues/247)).
- Fix a bug where dynamic variables (those declared with `sh:`) were not
  running in the task directory when the task has a custom dir or it was
  in an included Taskfile
  ([#384](https://github.com/go-task/task/issues/384)).
- The watch feature (via the `--watch` flag) got a few different bug fixes and
  should be more stable now
  ([#423](https://github.com/go-task/task/pull/423), [#365](https://github.com/go-task/task/issues/365)).

## v3.1.0

- Fix a bug when the checksum up-to-date resolution is used by a task
  with a custom `label:` attribute
  ([#412](https://github.com/go-task/task/issues/412)).
- Starting from this release, we're releasing official ARMv6 and ARM64 binaries
  for Linux
  ([#375](https://github.com/go-task/task/issues/375), [#418](https://github.com/go-task/task/issues/418)).
- Task now respects the order of declaration of included Taskfiles when
  evaluating variables declaring by them
  ([#393](https://github.com/go-task/task/issues/393)).
- `set -e` is now automatically set on every command. This was done to fix an
  issue where multiline string commands wouldn't really fail unless the
  sentence was in the last line
  ([#403](https://github.com/go-task/task/issues/403)).

## v3.0.1

- Allow use as a library by moving the required packages out of the `internal`
  directory
  ([#358](https://github.com/go-task/task/pull/358)).
- Do not error if a specified dotenv file does not exist
  ([#378](https://github.com/go-task/task/issues/378), [#385](https://github.com/go-task/task/pull/385)).
- Fix panic when you have empty tasks in your Taskfile
  ([#338](https://github.com/go-task/task/issues/338), [#362](https://github.com/go-task/task/pull/362)).

## v3.0.0

- On `v3`, all CLI variables will be considered global variables
  ([#336](https://github.com/go-task/task/issues/336), [#341](https://github.com/go-task/task/pull/341))
- Add support to `.env` like files
  ([#324](https://github.com/go-task/task/issues/324), [#356](https://github.com/go-task/task/pull/356)).
- Add `label:` to task so you can override the task name in the logs
  ([#321](https://github.com/go-task/task/issues/321]), [#337](https://github.com/go-task/task/pull/337)).
- Refactor how variables work on version 3
  ([#311](https://github.com/go-task/task/pull/311)).
- Disallow `expansions` on v3 since it has no effect.
- `Taskvars.yml` is not automatically included anymore.
- `Taskfile_{{OS}}.yml` is not automatically included anymore.
- Allow interpolation on `includes`, so you can manually include a Taskfile
  based on operation system, for example.
- Expose `.TASK` variable in templates with the task name
  ([#252](https://github.com/go-task/task/issues/252)).
- Implement short task syntax
  ([#194](https://github.com/go-task/task/issues/194), [#240](https://github.com/go-task/task/pull/240)).
- Added option to make included Taskfile run commands on its own directory
  ([#260](https://github.com/go-task/task/issues/260), [#144](https://github.com/go-task/task/issues/144))
- Taskfiles in version 1 are not supported anymore
  ([#237](https://github.com/go-task/task/pull/237)).
- Added global `method:` option. With this option, you can set a default
  method to all tasks in a Taskfile
  ([#246](https://github.com/go-task/task/issues/246)).
- Changed default method from `timestamp` to `checksum`
  ([#246](https://github.com/go-task/task/issues/246)).
- New magic variables are now available when using `status:`:
  `.TIMESTAMP` which contains the greatest modification date
  from the files listed in `sources:`, and `.CHECKSUM`, which
  contains a checksum of all files listed in `status:`.
  This is useful for manual checking when using external, or even remote,
  artifacts when using `status:`
  ([#216](https://github.com/go-task/task/pull/216)).
- We're now using [slim-sprig](https://github.com/go-task/slim-sprig) instead of
  [sprig](https://github.com/Masterminds/sprig), which allowed a file size
  reduction of about 22%
  ([#219](https://github.com/go-task/task/pull/219)).
- We now use some colors on Task output to better distinguish message types -
  commands are green, errors are red, etc
  ([#207](https://github.com/go-task/task/pull/207)).

## v2.8.1 - 2019-05-20

- Fix error code for the `--help` flag
  ([#300](https://github.com/go-task/task/issues/300), [#330](https://github.com/go-task/task/pull/330)).
- Print version to stdout instead of stderr
  ([#299](https://github.com/go-task/task/issues/299), [#329](https://github.com/go-task/task/pull/329)).
- Supress `context` errors when using the `--watch` flag
  ([#313](https://github.com/go-task/task/issues/313), [#317](https://github.com/go-task/task/pull/317)).
- Support templating on description
  ([#276](https://github.com/go-task/task/issues/276), [#283](https://github.com/go-task/task/pull/283)).

## v2.8.0 - 2019-12-07

- Add `--parallel` flag (alias `-p`) to run tasks given by the command line in
  parallel
  ([#266](https://github.com/go-task/task/pull/266)).
- Fixed bug where calling the `task` CLI only informing global vars would not
  execute the `default` task.
- Add hability to silent all tasks by adding `silent: true` a the root of the
  Taskfile.

## v2.7.1 - 2019-11-10

- Fix error being raised when `exit 0` was called
  ([#251](https://github.com/go-task/task/issues/251)).

## v2.7.0 - 2019-09-22

- Fixed panic bug when assigning a global variable
  ([#229](https://github.com/go-task/task/issues/229), [#243](https://github.com/go-task/task/issues/234)).
- A task with `method: checksum` will now re-run if generated files are deleted
  ([#228](https://github.com/go-task/task/pull/228), [#238](https://github.com/go-task/task/issues/238)).

## v2.6.0 - 2019-07-21

- Fixed some bugs regarding minor version checks on `version:`.
- Add `preconditions:` to task
  ([#205](https://github.com/go-task/task/pull/205)).
- Create directory informed on `dir:` if it doesn't exist
  ([#209](https://github.com/go-task/task/issues/209), [#211](https://github.com/go-task/task/pull/211)).
- We now have a `--taskfile` flag (alias `-t`), which can be used to run
  another Taskfile (other than the default `Taskfile.yml`)
  ([#221](https://github.com/go-task/task/pull/221)).
- It's now possible to install Task using Homebrew on Linux
  ([go-task/homebrew-tap#1](https://github.com/go-task/homebrew-tap/pull/1)).

## v2.5.2 - 2019-05-11

- Reverted YAML upgrade due issues with CRLF on Windows
  ([#201](https://github.com/go-task/task/issues/201), [go-yaml/yaml#450](https://github.com/go-yaml/yaml/issues/450)).
- Allow setting global variables through the CLI
  ([#192](https://github.com/go-task/task/issues/192)).

## 2.5.1 - 2019-04-27

- Fixed some issues with interactive command line tools, where sometimes
  the output were not being shown, and similar issues
  ([#114](https://github.com/go-task/task/issues/114), [#190](https://github.com/go-task/task/issues/190), [#200](https://github.com/go-task/task/pull/200)).
- Upgraded [go-yaml/yaml](https://github.com/go-yaml/yaml) from v2 to v3.

## v2.5.0 - 2019-03-16

- We moved from the taskfile.org domain to the new fancy taskfile.dev domain.
  While stuff is being redirected, we strongly recommend to everyone that use
  [this install script](https://taskfile.dev/#/installation?id=install-script)
  to use the new taskfile.dev domain on scripts from now on.
- Fixed to the ZSH completion
  ([#182](https://github.com/go-task/task/pull/182)).
- Add [`--summary` flag along with `summary:` task attribute](https://taskfile.org/#/usage?id=display-summary-of-task)
  ([#180](https://github.com/go-task/task/pull/180)).

## v2.4.0 - 2019-02-21

- Allow calling a task of the root Taskfile from an included Taskfile
  by prefixing it with `:`
  ([#161](https://github.com/go-task/task/issues/161), [#172](https://github.com/go-task/task/issues/172)),
- Add flag to override the `output` option
  ([#173](https://github.com/go-task/task/pull/173));
- Fix bug where Task was persisting the new checksum on the disk when the Dry
  Mode is enabled
  ([#166](https://github.com/go-task/task/issues/166));
- Fix file timestamp issue when the file name has spaces
  ([#176](https://github.com/go-task/task/issues/176));
- Mitigating path expanding issues on Windows
  ([#170](https://github.com/go-task/task/pull/170)).

## v2.3.0 - 2019-01-02

- On Windows, Task can now be installed using [Scoop](https://scoop.sh/)
  ([#152](https://github.com/go-task/task/pull/152));
- Fixed issue with file/directory globing
  ([#153](https://github.com/go-task/task/issues/153));
- Added ability to globally set environment variables
  (
    [#138](https://github.com/go-task/task/pull/138),
    [#159](https://github.com/go-task/task/pull/159)
  ).

## v2.2.1 - 2018-12-09

- This repository now uses Go Modules (#143). We'll still keep the `vendor` directory in sync for some time, though;
- Fixing a bug when the Taskfile has no tasks but includes another Taskfile (#150);
- Fix a bug when calling another task or a dependency in an included Taskfile (#151).

## v2.2.0 - 2018-10-25

- Added support for [including other Taskfiles](https://taskfile.org/#/usage?id=including-other-taskfiles) (#98)
  - This should be considered experimental. For now, only including local files is supported, but support for including remote Taskfiles is being discussed. If you have any feedback, please comment on #98.
- Task now have a dedicated documentation site: https://taskfile.org
  - Thanks to [Docsify](https://docsify.js.org/) for making this pretty easy. To check the source code, just take a look at the [docs](https://github.com/go-task/task/tree/master/docs) directory of this repository. Contributions to the documentation is really appreciated.

## v2.1.1 - 2018-09-17

- Fix suggestion to use `task --init` not being shown anymore (when a `Taskfile.yml` is not found)
- Fix error when using checksum method and no file exists for a source glob (#131)
- Fix signal handling when the `--watch` flag is given (#132)

## v2.1.0 - 2018-08-19

- Add a `ignore_error` option to task and command (#123)
- Add a dry run mode (`--dry` flag) (#126)

## v2.0.3 - 2018-06-24

- Expand environment variables on "dir", "sources" and "generates" (#116)
- Fix YAML merging syntax (#112)
- Add ZSH completion (#111)
- Implement new `output` option. Please check out the [documentation](https://github.com/go-task/task#output-syntax)

## v2.0.2 - 2018-05-01

- Fix merging of YAML anchors (#112)

## v2.0.1 - 2018-03-11

- Fixes panic on `task --list`

## v2.0.0 - 2018-03-08

Version 2.0.0 is here, with a new Taskfile format.

Please, make sure to read the [Taskfile versions](https://github.com/go-task/task/blob/master/TASKFILE_VERSIONS.md) document, since it describes in depth what changed for this version.

* New Taskfile version 2 (https://github.com/go-task/task/issues/77)
* Possibility to have global variables in the `Taskfile.yml` instead of `Taskvars.yml` (https://github.com/go-task/task/issues/66)
* Small improvements and fixes

## v1.4.4 - 2017-11-19

- Handle SIGINT and SIGTERM (#75);
- List: print message with there's no task with description;
- Expand home dir ("~" symbol) on paths (#74);
- Add Snap as an installation method;
- Move examples to its own repo;
- Watch: also walk on tasks called on on "cmds", and not only on "deps";
- Print logs to stderr instead of stdout (#68);
- Remove deprecated `set` keyword;
- Add checksum based status check, alternative to timestamp based.

## v1.4.3 - 2017-09-07

- Allow assigning variables to tasks at run time via CLI (#33)
- Added suport for multiline variables from sh (#64)
- Fixes env: remove square braces and evaluate shell (#62)
- Watch: change watch library and few fixes and improvements
- When use watching, cancel and restart long running process on file change (#59 and #60)

## v1.4.2 - 2017-07-30

- Flag to set directory of execution
- Always echo command if is verbose mode
- Add silent mode to disable echoing of commands
- Fixes and improvements of variables (#56)

## v1.4.1 - 2017-07-15

- Allow use of YAML for dynamic variables instead of $ prefix
  - `VAR: {sh: echo Hello}` instead of `VAR: $echo Hello`
- Add `--list` (or `-l`) flag to print existing tasks
- OS specific Taskvars file (e.g. `Taskvars_windows.yml`, `Taskvars_linux.yml`, etc)
- Consider task up-to-date on equal timestamps (#49)
- Allow absolute path in generates section (#48)
- Bugfix: allow templating when calling deps (#42)
- Fix panic for invalid task in cyclic dep detection
- Better error output for dynamic variables in Taskvars.yml (#41)
- Allow template evaluation in parameters

## v1.4.0 - 2017-07-06

- Cache dynamic variables
- Add verbose mode (`-v` flag)
- Support to task parameters (overriding vars) (#31) (#32)
- Print command, also when "set:" is specified (#35)
- Improve task command help text (#35)

## v1.3.1 - 2017-06-14

- Fix glob not working on commands (#28)
- Add ExeExt template function
- Add `--init` flag to create a new Taskfile
- Add status option to prevent task from running (#27)
- Allow interpolation on `generates` and `sources` attributes (#26)

## v1.3.0 - 2017-04-24

- Migrate from os/exec.Cmd to a native Go sh/bash interpreter
  - This is a potentially breaking change if you use Windows.
  - Now, `cmd` is not used anymore on Windows. Always use Bash-like syntax for your commands, even on Windows.
- Add "ToSlash" and "FromSlash" to template functions
- Use functions defined on github.com/Masterminds/sprig
- Do not redirect stdin while running variables commands
- Using `context` and `errgroup` packages (this will make other tasks to be cancelled, if one returned an error)

## v1.2.0 - 2017-04-02

- More tests and Travis integration
- Watch a task (experimental)
- Possibility to call another task
- Fix "=" not being reconized in variables/environment variables
- Tasks can now have a description, and help will print them (#10)
- Task dependencies now run concurrently
- Support for a default task (#16)

## v1.1.0 - 2017-03-08

- Support for YAML, TOML and JSON (#1)
- Support running command in another directory (#4)
- `--force` or `-f` flag to force execution of task even when it's up-to-date
- Detection of cyclic dependencies (#5)
- Support for variables (#6, #9, #14)
- Operation System specific commands and variables (#13)

## v1.0.0 - 2017-02-28

- Add LICENSE file
