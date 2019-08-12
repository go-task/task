# Changelog

## Unreleased - v3

- We're now using [slim-sprig](https://github.com/go-task/slim-sprig) instead of
  [sprig](https://github.com/Masterminds/sprig), which allowed a file size
  reduction of about 22%
  ([#219](https://github.com/go-task/task/pull/219)).
- We now use some colors on Task output to better distinguish message types -
  commands are green, errors are red, etc
  ([#207](https://github.com/go-task/task/pull/207)).

## Unreleased

- Fixed panic bug when assigning a global variable
  ([#229](https://github.com/go-task/task/issues/229), [#243](https://github.com/go-task/task/issues/234)).

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
