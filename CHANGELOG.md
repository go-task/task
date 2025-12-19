# Changelog

## v3.46.3 - 2025-12-19

- Fixed regression in completion script for zsh (#2593, #2594 by @vmaerten).

## v3.46.2 - 2025-12-18

- Fixed a regression on previous release that affected variables passed via
  command line (#2588, #2589 by @vmaerten).

## v3.46.1 - 2025-12-18

### ✨ Features

- A small behavior change was made to dependencies. Task will now wait for all
  dependencies to finish running before continuing, even if any of them fail. To
  opt for the previous behavior, set `failfast: true` either on your
  `.taskrc.yml` or per task, or use the `--failfast` flag, which will also work
  for `--parallel` (#1246, #2525 by @andreynering).
- The `--summary` flag now displays `vars:` (both global and task-level),
  `env:`, and `requires:` sections. Dynamic variables show their shell command
  (e.g., `sh: echo "hello"`) instead of the evaluated value (#2486 ,#2524 by
  @vmaerten).
- Improved performance of fuzzy task name matching by implementing lazy
  initialization. Added `--disable-fuzzy` flag and `disable-fuzzy` taskrc option
  to allow disabling fuzzy matching entirely (#2521, #2523 by @vmaerten).
- Added LLM-optimized documentation via VitePress plugin, generating `llms.txt`
  and `llms-full.txt` for AI-powered development tools (#2513 by @vmaerten).
- Added `--trusted-hosts` CLI flag and `remote.trusted-hosts` config option to
  skip confirmation prompts for specified hosts when using Remote Taskfiles
  (#2491, #2473 by @maciejlech).
- When running in GitHub Actions, Task now automatically emits error annotations
  on failure, improving visibility in workflow summaries (#2568 by @vmaerten).
- The `--yes` flag is now accessible in templates via the new `CLI_ASSUME_YES`
  variable (#2577, #2479 by @semihbkgr).
- Improved shell completion scripts (Zsh, Fish, PowerShell) by adding missing
  flags and dynamic experimental feature detection (#2532 by @vmaerten).
- Remote Taskfiles now accept `application/octet-stream` Content-Type (#2536,
  #1944 by @vmaerten).
- Shell completion now works when Task is installed or aliased under a different
  binary name via TASK_EXE environment variable (#2495, #2468 by @vmaerten).
- Some small fixes and improvements were made to `task --init` and to the
  default Taskfile it generates (#2433 by @andreynering).
- Added `--remote-cache-dir` flag and `remote.cache-dir` taskrc option to
  customize the cache directory for Remote Taskfiles (#2572 by @vmaerten).
- Zsh completion now supports zstyle verbose option to show or hide task
  descriptions (#2571 by @vmaerten).
- Task now automatically enables colored output in CI environments (GitHub
  Actions, GitLab CI, etc.) without requiring FORCE_COLOR=1 (#2569 by
  @vmaerten).
- Added color taskrc option to explicitly enable or disable colored output
  globally (#2569 by @vmaerten).
- Improved Git Remote Taskfiles by switching to go-getter: SSH authentication
  now works out of the box and `applyOf` is properly supported (#2512 by
  @vmaerten).

### 🐛 Fixes

- Fix RPM upload to Cloudsmith by including the version in the filename to
  ensure unique filenames (#2507 by @vmaerten).
- Fix `run: when_changed` to work properly for Taskfiles included multiple times
  (#2508, #2511 by @trulede).
- Fixed Zsh and Fish completions to stop suggesting task names after `--`
  separator, allowing proper CLI_ARGS completion (#1843, #1844 by
  @boiledfroginthewell).
- Watch mode (`--watch`) now always runs the task, regardless of `run: once` or
  `run: when_changed` settings (#2566, #1388 by @trulede).
- Fixed global variables (CLI_ARGS, CLI_FORCE, etc.) not being accessible in
  root-level vars section (#2403, #2397 by @trulede, @vmaerten).
- Fixed a bug where `ignore_error` was ignored when using `task:` to call
  another task (#2552, #363 by @trulede).
- Fixed Zsh completion not suggesting global tasks when using `-g`/`--global`
  flag (#1574, #2574 by @vmaerten).
- Fixed Fish completion failing to parse task descriptions containing colons
  (e.g., URLs or namespaced functions) (#2101, #2573 by @vmaerten).
- Fixed false positive "property 'for' is not allowed" warnings in IntelliJ when
  using `for` loops in Taskfiles (#2576 by @vmaerten).

## v3.45.5 - 2025-11-11

- Fixed bug that made a generic message, instead of an useful one, appear when a
  Taskfile could not be found (#2431 by @andreynering).
- Fixed a bug that caused an error when including a Remote Git Taskfile (#2438
  by @twelvelabs).
- Fixed issue where `.taskrc.yml` was not returned if reading it failed, and
  corrected handling of remote entrypoint Taskfiles (#2460, #2461 by @vmaerten).
- Improved performance of `--list` and `--list-all` by introducing a faster
  compilation method that skips source globbing and checksum updates (#1322,
  #2053 by @vmaerten).
- Fixed a concurrency bug with `output: group`. This ensures that begin/end
  parts won't be mixed up from different tasks (#1208, #2349, #2350 by
  @trulede).
- Do not re-evaluate variables for `defer:` (#2244, #2418 by @trulede).
- Improve error message when a Taskfile is not found (#2441, #2494 by
  @vmaerten).
- Fixed generic error message `exit status 1` when a dependency task failed
  (#2286 by @GrahamDennis).
- Fixed YAML library from the unmaintained `gopkg.in/yaml.v3` to the new fork
  maintained by the official YAML org (#2171, #2434 by @andreynering).
- On Windows, the built-in version of the `rm` core utils contains a fix related
  to the `-f` flag (#2426,
  [u-root/u-root#3464](https://github.com/u-root/u-root/pull/3464),
  [mvdan/sh#1199](https://github.com/mvdan/sh/pull/1199), #2506 by
  @andreynering).

## v3.45.4 - 2025-09-17

- Fixed a bug where `cache-expiry` could not be defined in `.taskrc.yml` (#2423
  by @vmaerten).
- Fixed a bug where `.taskrc.yml` files in parent folders were not read
  correctly (#2424 by @vmaerten).
- Fixed a bug where autocomplete in subfolders did not work with zsh (#2425 by
  @vmaerten).

## v3.45.3 - 2025-09-15

- Task now includes built-in core utilities to greatly improve compatibility on
  Windows. This means that your commands that uses `cp`, `mv`, `mkdir` or any
  other common core utility will now work by default on Windows, without extra
  setup. This is something we wanted to address for many many years, and it's
  finally being shipped!
  [Read our blog post this the topic](https://taskfile.dev/blog/windows-core-utils).
  (#197, #2360 by @andreynering).
- :sparkles: Built and deployed a [brand new website](https://taskfile.dev)
  using [VitePress](https://vitepress.dev) (#2359, #2369, #2371, #2375, #2378 by
  @vmaerten, @andreynering, @pd93).
- Began releasing
  [nightly builds](https://github.com/go-task/task/releases/tag/nightly). This
  will allow people to test our changes before they are fully released and
  without having to install Go to build them (#2358 by @vmaerten).
- Added support for global config files in `$XDG_CONFIG_HOME/task/taskrc.yml` or
  `$HOME/.taskrc.yml`. Check out our new
  [configuration guide](https://taskfile.dev/docs/reference/config) for more
  details (#2247, #2380, #2390, #2391 by @vmaerten, @pd93).
- Added experiments to the taskrc schema to clarify the expected keys and values
  (#2235 by @vmaerten).
- Added support for new properties in `.taskrc.yml`: insecure, verbose,
  concurrency, remote offline, remote timeout, and remote expiry. :warning:
  Note: setting offline via environment variable is no longer supported. (#2389
  by @vmaerten)
- Added a `--nested` flag when outputting tasks using `--list --json`. This will
  output tasks in a nested structure when tasks are namespaced (#2415 by @pd93).
- Enhanced support for tasks with wildcards: they are now logged correctly, and
  wildcard parameters are fully considered during fingerprinting (#1808, #1795
  by @vmaerten).
- Fixed panic when a variable was declared as an empty hash (`{}`) (#2416, #2417
  by @trulede).

#### Package API

- Bumped the minimum version of Go to 1.24 (#2358 by @vmaerten).

#### Other news

We recently released our
[official GitHub Action](https://github.com/go-task/setup-task). This is based
on the fantastic work by the Arduino team who created and maintained the
community version. Now that this is officially adopted, fixes/updates should be
more timely. We have already merged a couple of longstanding PRs in our
[first release](https://github.com/go-task/setup-task/releases/tag/v1.0.0) (by
@pd93, @shrink, @trim21 and all the previous contributors to
[arduino/setup-task](https://github.com/arduino/setup-task/)).

## v3.45.0-v3.45.2 - 2025-09-15

Failed due to an issue with our release process.

## v3.44.1 - 2025-07-23

- Internal tasks will no longer be shown as suggestions since they cannot be
  called (#2309, #2323 by @maxmzkrcensys)
- Fixed install script for some ARM platforms (#1516, #2291 by @trulede).
- Fixed a regression where fingerprinting was not working correctly if the path
  to you Taskfile contained a space (#2321, #2322 by @pd93).
- Reverted a breaking change to `randInt` (#2312, #2316 by @pd93).
- Made new variables `TEST_NAME` and `TEST_DIR` available in fixture tests
  (#2265 by @pd93).

## v3.44.0 - 2025-06-08

- Added `uuid`, `randInt` and `randIntN` template functions (#1346, #2225 by
  @pd93).
- Added new `CLI_ARGS_LIST` array variable which contains the arguments passed
  to Task after the `--` (the same as `CLI_ARGS`, but an array instead of a
  string). (#2138, #2139, #2140 by @pd93).
- Added `toYaml` and `fromYaml` templating functions (#2217, #2219 by @pd93).
- Added `task` field the `--list --json` output (#2256 by @aleksandersh).
- Added the ability to
  [pin included taskfiles](https://taskfile.dev/next/experiments/remote-taskfiles/#manual-checksum-pinning)
  by specifying a checksum. This works with both local and remote Taskfiles
  (#2222, #2223 by @pd93).
- When using the
  [Remote Taskfiles experiment](https://github.com/go-task/task/issues/1317),
  any credentials used in the URL will now be redacted in Task's output (#2100,
  #2220 by @pd93).
- Fixed fuzzy suggestions not working when misspelling a task name (#2192, #2200
  by @vmaerten).
- Fixed a bug where taskfiles in directories containing spaces created
  directories in the wrong location (#2208, #2216 by @pd93).
- Added support for dual JSON schema files, allowing changes without affecting
  the current schema. The current schemas will only be updated during releases.
  (#2211 by @vmaerten).
- Improved fingerprint documentation by specifying that the method can be set at
  the root level to apply to all tasks (#2233 by @vmaerten).
- Fixed some watcher regressions after #2048 (#2199, #2202, #2241, #2196 by
  @wazazaby, #2271 by @andreynering).

## v3.43.3 - 2025-04-27

Reverted the changes made in #2113 and #2186 that affected the
`USER_WORKING_DIR` and built-in variables. This fixes #2206, #2195, #2207 and
#2208.

## v3.43.2 - 2025-04-21

- Fixed regresion of `CLI_ARGS` being exposed as the wrong type (#2190, #2191 by
  @vmaerten).

## v3.43.1 - 2025-04-21

- Significant improvements were made to the watcher. We migrated from
  [watcher](https://github.com/radovskyb/watcher) to
  [fsnotify](https://github.com/fsnotify/fsnotify). The former library used
  polling, which means Task had a high CPU usage when watching too many files.
  `fsnotify` uses proper the APIs from each operating system to watch files,
  which means a much better performance. The default interval changed from 5
  seconds to 100 milliseconds, because now it configures the wait time for
  duplicated events, instead of the polling time (#2048 by @andreynering, #1508,
  #985, #1179).
- The [Map Variables experiment](https://github.com/go-task/task/issues/1585)
  was made generally available so you can now
  [define map variables in your Taskfiles!](https://taskfile.dev/usage/#variables)
  (#1585, #1547, #2081 by @pd93).
- Wildcards can now
  [match multiple tasks](https://taskfile.dev/usage/#wildcard-arguments) (#2072,
  #2121 by @pd93).
- Added the ability to
  [loop over the files specified by the `generates` keyword](https://taskfile.dev/usage/#looping-over-your-tasks-sources-or-generated-files).
  This works the same way as looping over sources (#2151 by @sedyh).
- Added the ability to resolve variables when defining an include variable
  (#2108, #2113 by @pd93).
- A few changes have been made to the
  [Remote Taskfiles experiment](https://github.com/go-task/task/issues/1317)
  (#1402, #2176 by @pd93):
  - Cached files are now prioritized over remote ones.
  - Added an `--expiry` flag which sets the TTL for a remote file cache. By
    default the value will be 0 (caching disabled). If Task is running in
    offline mode or fails to make a connection, it will fallback on the cache.
- `.taskrc` files can now be used from subdirectories and will be searched for
  recursively up the file tree in the same way that Taskfiles are (#2159, #2166
  by @pd93).
- The default taskfile (output when using the `--init` flag) is now an embedded
  file in the binary instead of being stored in the code (#2112 by @pd93).
- Improved the way we report the Task version when using the `--version` flag or
  `{{.TASK_VERSION}}` variable. This should now be more consistent and easier
  for package maintainers to use (#2131 by @pd93).
- Fixed a bug where globstar (`**`) matching in `sources` only resolved the
  first result (#2073, #2075 by @pd93).
- Fixed a bug where sorting tasks by "none" would use the default sorting
  instead of leaving tasks in the order they were defined (#2124, #2125 by
  @trulede).
- Fixed Fish completion on newer Fish versions (#2130 by @atusy).
- Fixed a bug where undefined/null variables resolved to an empty string instead
  of `nil` (#1911, #2144 by @pd93).
- The `USER_WORKING_DIR` special now will now properly account for the `--dir`
  (`-d`) flag, if given (#2102, #2103 by @jaynis, #2186 by @andreynering).
- Fix Fish completions when `--global` (`-g`) is given (#2134 by @atusy).
- Fixed variables not available when using `defer:` (#1909, #2173 by @vmaerten).

#### Package API

- The [`Executor`](https://pkg.go.dev/github.com/go-task/task/v3#Executor) now
  uses the functional options pattern (#2085, #2147, #2148 by @pd93).
- The functional options for the
  [`taskfile.Reader`](https://pkg.go.dev/github.com/go-task/task/v3/taskfile#Reader)
  and
  [`taskfile.Snippet`](https://pkg.go.dev/github.com/go-task/task/v3/taskfile#Snippet)
  types no longer have the `Reader`/`Snippet` respective prefixes (#2148 by
  @pd93).
- [`taskfile.Reader`](https://pkg.go.dev/github.com/go-task/task/v3/taskfile#Reader)
  no longer accepts a
  [`taskfile.Node`](https://pkg.go.dev/github.com/go-task/task/v3/taskfile#Node).
  Instead nodes are passed directly into the
  [`Reader.Read`](https://pkg.go.dev/github.com/go-task/task/v3/taskfile#Reader.Read)
  method (#2169 by @pd93).
- [`Reader.Read`](https://pkg.go.dev/github.com/go-task/task/v3/taskfile#Reader.Read)
  also now accepts a [`context.Context`](https://pkg.go.dev/context#Context)
  (#2176 by @pd93).

## v3.42.1 - 2025-03-10

- Fixed a bug where some special variables caused a type error when used global
  variables (#2106, #2107 by @pd93).

## v3.42.0 - 2025-03-08

- Made `--init` less verbose by default and respect `--silent` and `--verbose`
  flags (#2009, #2011 by @HeCorr).
- `--init` now accepts a file name or directory as an argument (#2008, #2018 by
  @HeCorr).
- Fix a bug where an HTTP node's location was being mutated incorrectly (#2007
  by @jeongukjae).
- Fixed a bug where allowed values didn't work with dynamic var (#2032, #2033 by
  @vmaerten).
- Use only the relevant checker (timestamp or checksum) to improve performance
  (#2029, #2031 by @vmaerten).
- Print warnings when attempting to enable an inactive experiment or an active
  experiment with an invalid value (#1979, #2049 by @pd93).
- Refactored the experiments package and added tests (#2049 by @pd93).
- Show allowed values when a variable with an enum is missing (#2027, #2052 by
  @vmaerten).
- Refactored how snippets in error work and added tests (#2068 by @pd93).
- Fixed a bug where errors decoding commands were sometimes unhelpful (#2068 by
  @pd93).
- Fixed a bug in the Taskfile schema where `defer` statements in the shorthand
  `cmds` syntax were not considered valid (#2068 by @pd93).
- Refactored how task sorting functions work (#1798 by @pd93).
- Added a new `.taskrc.yml` (or `.taskrc.yaml`) file to let users enable
  experiments (similar to `.env`) (#1982 by @vmaerten).
- Added new [Getting Started docs](https://taskfile.dev/getting-started) (#2086
  by @pd93).
- Allow `matrix` to use references to other variables (#2065, #2069 by @pd93).
- Fixed a bug where, when a dynamic variable is provided, even if it is not
  used, all other variables become unavailable in the templating system within
  the include (#2092 by @vmaerten).

#### Package API

Unlike our CLI tool,
[Task's package API is not currently stable](https://taskfile.dev/reference/package).
In an effort to ease the pain of breaking changes for our users, we will be
providing changelogs for our package API going forwards. The hope is that these
changes will provide a better long-term experience for our users and allow to
stabilize the API in the future. #121 now tracks this piece of work.

- Bumped the minimum required Go version to 1.23 (#2059 by @pd93).
- [`task.InitTaskfile`](https://pkg.go.dev/github.com/go-task/task/v3#InitTaskfile)
  (#2011, ff8c913 by @HeCorr and @pd93)
  - No longer accepts an `io.Writer` (output is now the caller's
    responsibility).
  - The path argument can now be a filename OR a directory.
  - The function now returns the full path of the generated file.
- [`TaskfileDecodeError.WithFileInfo`](https://pkg.go.dev/github.com/go-task/task/v3/errors#TaskfileDecodeError.WithFileInfo)
  now accepts a string instead of the arguments required to generate a snippet
  (#2068 by @pd93).
  - The caller is now expected to create the snippet themselves (see below).
- [`TaskfileSnippet`](https://pkg.go.dev/github.com/go-task/task/v3/taskfile#Snippet)
  and related code moved from the `errors` package to the `taskfile` package
  (#2068 by @pd93).
- Renamed `TaskMissingRequiredVars` to
  [`TaskMissingRequiredVarsError`](https://pkg.go.dev/github.com/go-task/task/v3/errors#TaskMissingRequiredVarsError)
  (#2052 by @vmaerten).
- Renamed `TaskNotAllowedVars` to
  [`TaskNotAllowedVarsError`](https://pkg.go.dev/github.com/go-task/task/v3/errors#TaskNotAllowedVarsError)
  (#2052 by @vmaerten).
- The
  [`taskfile.Reader`](https://pkg.go.dev/github.com/go-task/task/v3/taskfile#Reader)
  is now constructed using the functional options pattern (#2082 by @pd93).
- Removed our internal `logger.Logger` from the entire `taskfile` package (#2082
  by @pd93).
  - Users are now expected to pass a custom debug/prompt functions into
    [`taskfile.Reader`](https://pkg.go.dev/github.com/go-task/task/v3/taskfile#Reader)
    if they want this functionality by using the new
    [`WithDebugFunc`](https://pkg.go.dev/github.com/go-task/task/v3/taskfile#WithDebugFunc)
    and
    [`WithPromptFunc`](https://pkg.go.dev/github.com/go-task/task/v3/taskfile#WithPromptFunc)
    functional options.
- Remove `Range` functions in the `taskfile/ast` package in favour of new
  iterator functions (#1798 by @pd93).
- `ast.Call` was moved from the `taskfile/ast` package to the main `task`
  package (#2084 by @pd93).
- `ast.Tasks.FindMatchingTasks` was moved from the `taskfile/ast` package to the
  `task.Executor.FindMatchingTasks` in the main `task` package (#2084 by @pd93).
- The `Compiler` and its `GetVariables` and `FastGetVariables` methods were
  moved from the `internal/compiler` package to the main `task` package (#2084
  by @pd93).

## v3.41.0 - 2025-01-18

- Fixed an issue where dynamic variables were not properly logged in verbose
  mode (#1920, #1921 by @mgbowman).
- Support `silent` for defer statements (#1877, #1879 by @danilobuerger).
- Added an option to exclude some tasks from being included (#1859 by
  @vmaerten).
- Fixed an issue where a required variable was incorrectly handled in a template
  function (#1950, #1962 by @vmaerten).
- Expose a new `TASK_DIR` special variable, which will contain the absolute path
  of task directory. (#1959, #1961 by @vmaerten).
- Fixed fatal bugs that caused concurrent map writes (#1605, #1972, #1974 by
  @pd93, @GrahamDennis and @trim21).
- Refactored internal ordered map implementation to use
  [github.com/elliotchance/orderedmap](https://github.com/elliotchance/orderedmap)
  (#1797 by @pd93).
- Fixed a bug where variables defined at the task level were being ignored in
  the `requires` section. (#1960, #1955, #1768 by @vmaerten and @mokeko)
- The `CHECKSUM` and `TIMESTAMP` variables are now accessible within `cmds`
  (#1872 by @niklasr22).
- Updated [installation docs](https://taskfile.dev/installation) and added pip
  installation method (#935, #1989 by @pd93).
- Fixed a bug where dynamic variables could not access environment variables
  (#630, #1869 by @rohm1 and @pd93).
- Disable version check for use as an external library (#1938 by @leaanthony).

## v3.40.1 - 2024-12-06

- Fixed a security issue in `git-urls` by switching to the maintained fork
  `chainguard-dev/git-urls` (#1917 by @AlekSi).
- Added missing `platforms` property to `cmds` that use `for` (#1915 by
  @dkarter).
- Added misspell linter to check for misspelled English words (#1883 by
  @christiandins).

## v3.40.0 - 2024-11-05

- Fixed output of some functions (e.g. `splitArgs`/`splitLines`) not working in
  for loops (#1822, #1823 by @stawii).
- Added a new `TASK_OFFLINE` environment variable to configure the `--offline`
  flag and expose it as a special variable in the templating system (#1470,
  #1716 by @vmaerten and @pd93).
- Fixed a bug where multiple remote includes caused all prompts to display
  without waiting for user input (#1832, #1833 by @vmaerten and @pd93).
- When using the
  "[Remote Taskfiles](https://taskfile.dev/experiments/remote-taskfiles/)".
  experiment, you can now include Taskfiles from Git repositories (#1652 by
  @vmaerten).
- Improved the error message when a dotenv file cannot be parsed (#1842 by
  @pbitty).
- Fix issue with directory when using the remote experiment (#1757 by @pbitty).
- Fixed an issue where a special variable was used in combination with a dotenv
  file (#1232, #1810 by @vmaerten).
- Refactor the way Task reads Taskfiles to improve readability (#1771 by
  @pbitty).
- Added a new option to ensure variable is within the list of values (#1827 by
  @vmaerten).
- Allow multiple prompts to be specified for a task (#1861, #1866 by @mfbmina).
- Added new template function: `numCPU`, which returns the number of logical
  CPUs usable (#1890, #1887 by @Amoghrd).
- Fixed a bug where non-nil, empty dynamic variables are returned as an empty
  interface (#1903, #1904 by @pd93).

## v3.39.2 - 2024-09-19

- Fix dynamic variables not working properly for a defer: statement (#1803,
  #1818 by @vmaerten).

## v3.39.1 - 2024-09-18

- Added Renovate configuration to automatically create PRs to keep dependencies
  up to date (#1783 by @vmaerten).
- Fixed a bug where the help was displayed twice (#1805, #1806 by @vmaerten).
- Fixed a bug where ZSH and PowerShell completions did not work when using the
  recommended method. (#1813, #1809 by @vmaerten and @shirayu)
- Fix variables not working properly for a `defer:` statement (#1803, #1814 by
  @vmaerten and @andreynering).

## v3.39.0 - 2024-09-07

- Added
  [Env Precedence Experiment](https://taskfile.dev/experiments/env-precedence)
  (#1038, #1633 by @vmaerten).
- Added a CI lint job to ensure that the docs are updated correctly (#1719 by
  @vmaerten).
- Updated minimum required Go version to 1.22 (#1758 by @pd93).
- Expose a new `EXIT_CODE` special variable on `defer:` when a command finishes
  with a non-zero exit code (#1484, #1762 by @dorimon-1 and @andreynering).
- Expose a new `ALIAS` special variable, which will contain the alias used to
  call the current task. Falls back to the task name. (#1764 by @DanStory).
- Fixed `TASK_REMOTE_DIR` environment variable not working when the path was
  absolute. (#1715 by @vmaerten).
- Added an option to declare an included Taskfile as flattened (#1704 by
  @vmaerten).
- Added a new
  [`--completion` flag](https://taskfile.dev/installation/#setup-completions) to
  output completion scripts for various shells (#293, #1157 by @pd93).
  - This is now the preferred way to install completions.
  - The completion scripts in the `completion` directory
    [are now deprecated](https://taskfile.dev/deprecations/completion-scripts/).
- Added the ability to
  [loop over a matrix of values](https://taskfile.dev/usage/#looping-over-a-matrix)
  (#1766, #1767, #1784 by @pd93).
- Fixed a bug in fish completion where aliases were not displayed (#1781, #1782
  by @vmaerten).
- Fixed panic when having a flattened included Taskfile that contains a
  `default` task (#1777, #1778 by @vmaerten).
- Optimized file existence checks for remote Taskfiles (#1713 by @vmaerten).

## v3.38.0 - 2024-06-30

- Added `TASK_EXE` special variable (#1616, #1624 by @pd93 and @andreynering).
- Some YAML parsing errors will now show in a more user friendly way (#1619 by
  @pd93).
- Prefixed outputs will now be colorized by default (#1572 by
  @AlexanderArvidsson)
- [References](https://taskfile.dev/usage/#referencing-other-variables) are now
  generally available (no experiments required) (#1654 by @pd93).
- Templating functions can now be used in references (#1645, #1654 by @pd93).
- Added a new
  [templating reference page](https://taskfile.dev/reference/templating/) to the
  documentation (#1614, #1653 by @pd93).
- If using the
  [Map Variables experiment (1)](https://taskfile.dev/experiments/map-variables/?proposal=1),
  references are available by
  [prefixing a string with a `#`](https://taskfile.dev/experiments/map-variables/?proposal=1#references)
  (#1654 by @pd93).
- If using the
  [Map Variables experiment (2)](https://taskfile.dev/experiments/map-variables/?proposal=2),
  the `yaml` and `json` keys are no longer available (#1654 by @pd93).
- Added a new `TASK_REMOTE_DIR` environment variable to configure where cached
  remote Taskfiles are stored (#1661 by @vmaerten).
- Added a new `--clear-cache` flag to clear the cache of remote Taskfiles (#1639
  by @vmaerten).
- Improved the readability of cached remote Taskfile filenames (#1636 by
  @vmaerten).
- Starting releasing a binary for the `riscv64` architecture on Linux (#1699 by
  @mengzhuo).
- Added `CLI_SILENT` and `CLI_VERBOSE` variables (#1480, #1669 by @Vince-Smith).
- Fixed a couple of bugs with the `prompt:` feature (#1657 by @pd93).
- Fixed JSON Schema to disallow invalid properties (#1657 by @pd93).
- Fixed version checks not working as intended (#872, #1663 by @vmaerten).
- Fixed a bug where included tasks were run multiple times even if `run: once`
  was set (#852, #1655 by @pd93).
- Fixed some bugs related to column formatting in the terminal (#1350, #1637,
  #1656 by @vmaerten).

## v3.37.2 - 2024-05-12

- Fixed a bug where an empty Taskfile would cause a panic (#1648 by @pd93).
- Fixed a bug where includes Taskfile variable were not being merged correctly
  (#1643, #1649 by @pd93).

## v3.37.1 - 2024-05-09

- Fix bug where non-string values (numbers, bools) added to `env:` weren't been
  correctly exported (#1640, #1641 by @vmaerten and @andreynering).

## v3.37.0 - 2024-05-08

- Released the
  [Any Variables experiment](https://taskfile.dev/blog/any-variables), but
  [_without support for maps_](https://github.com/go-task/task/issues/1415#issuecomment-2044756925)
  (#1415, #1547 by @pd93).
- Refactored how Task reads, parses and merges Taskfiles using a DAG (#1563,
  #1607 by @pd93).
- Fix a bug which stopped tasks from using `stdin` as input (#1593, #1623 by
  @pd93).
- Fix error when a file or directory in the project contained a special char
  like `&`, `(` or `)` (#1551, #1584 by @andreynering).
- Added alias `q` for template function `shellQuote` (#1601, #1603 by @vergenzt)
- Added support for `~` on ZSH completions (#1613 by @jwater7).
- Added the ability to pass variables by reference using Go template syntax when
  the
  [Map Variables experiment](https://taskfile.dev/experiments/map-variables/) is
  enabled (#1612 by @pd93).
- Added support for environment variables in the templating engine in `includes`
  (#1610 by @vmaerten).

## v3.36.0 - 2024-04-08

- Added support for
  [looping over dependencies](https://taskfile.dev/usage/#looping-over-dependencies)
  (#1299, #1541 by @pd93).
- When using the
  "[Remote Taskfiles](https://taskfile.dev/experiments/remote-taskfiles/)"
  experiment, you are now able to use
  [remote Taskfiles as your entrypoint](https://taskfile.dev/experiments/remote-taskfiles/#root-remote-taskfiles).
  - `includes` in remote Taskfiles will now also resolve correctly (#1347 by
    @pd93).
- When using the
  "[Any Variables](https://taskfile.dev/experiments/any-variables/)"
  experiments, templating is now supported in collection-type variables (#1477,
  #1511, #1526 by @pd93).
- Fixed a bug where variables being passed to an included Taskfile were not
  available when defining global variables (#1503, #1533 by @pd93).
- Improved support to customized colors by allowing 8-bit colors and multiple
  ANSI attributes (#1576 by @pd93).

## v3.35.1 - 2024-03-04

- Fixed a bug where the `TASKFILE_DIR` variable was sometimes incorrect (#1522,
  #1523 by @pd93).
- Added a new `TASKFILE` special variable that holds the root Taskfile path
  (#1523 by @pd93).
- Fixed various issues related to running a Taskfile from a subdirectory (#1529,
  #1530 by @pd93).

## v3.35.0 - 2024-02-28

- Added support for
  [wildcards in task names](https://taskfile.dev/usage/#wildcard-arguments)
  (#836, #1489 by @pd93).
- Added the ability to
  [run Taskfiles via stdin](https://taskfile.dev/usage/#reading-a-taskfile-from-stdin)
  (#655, #1483 by @pd93).
- Bumped minimum Go version to 1.21 (#1500 by @pd93).
- Fixed bug related to the `--list` flag (#1509, #1512 by @pd93, #1514, #1520 by
  @pd93).
- Add mention on the documentation to the fact that the variable declaration
  order is respected (#1510 by @kirkrodrigues).
- Improved style guide docs (#1495 by @iwittkau).
- Removed duplicated entry for `requires` on the API docs (#1491 by
  @teatimeguest).

## v3.34.1 - 2024-01-27

- Fixed prompt regression on
  [Remote Taskfiles experiment](https://taskfile.dev/experiments/remote-taskfiles/)
  (#1486, #1487 by @pd93).

## v3.34.0 - 2024-01-25

- Removed support for `version: 2` schemas. See the
  [deprecation notice on our website](https://taskfile.dev/deprecations/version-2-schema)
  (#1197, #1447 by @pd93).
- Fixed a couple of issues in the JSON Schema + added a CI step to ensure it's
  correct (#1471, #1474, #1476 by @sirosen).
- Added
  [Any Variables experiment proposal 2](https://taskfile.dev/experiments/any-variables/?proposal=2)
  (#1415, #1444 by @pd93).
- Updated the experiments and deprecations documentation format (#1445 by
  @pd93).
- Added new template function: `spew`, which can be used to print variables for
  debugging purposes (#1452 by @pd93).
- Added new template function: `merge`, which can be used to merge any number of
  map variables (#1438, #1464 by @pd93).
- Small change on the API when using as a library: `call.Direct` became
  `call.Indirect` (#1459 by @pd93).
- Refactored the public `read` and `taskfile` packages and introduced
  `taskfile/ast` (#1450 by @pd93).
- `ast.IncludedTaskfiles` renamed to `ast.Includes` and `orderedmap` package
  renamed to `omap` plus some internal refactor work (#1456 by @pd93).
- Fix zsh completion script to allow lowercase `taskfile` file names (#1482 by
  @xontab).
- Improvements on how we check the Taskfile version (#1465 by @pd93).
- Added a new `ROOT_TASKFILE` special variable (#1468, #1469 by @pd93).
- Fix experiment flags in `.env` when the `--dir` or `--taskfile` flags were
  used (#1478 by @pd93).

## v3.33.1 - 2023-12-21

- Added support for looping over map variables with the
  [Any Variables experiment](https://taskfile.dev/experiments/any-variables)
  enabled (#1435, #1437 by @pd93).
- Fixed a bug where dynamic variables were causing errors during fast
  compilation (#1435, #1437 by @pd93)

## v3.33.0 - 2023-12-20

- Added
  [Any Variables experiment](https://taskfile.dev/experiments/any-variables)
  (#1415, #1421 by @pd93).
- Updated Docusaurus to v3 (#1432 by @pd93).
- Added `aliases` to `--json` flag output (#1430, #1431 by @pd93).
- Added new `CLI_FORCE` special variable containing whether the `--force` or
  `--force-all` flags were set (#1412, #1434 by @pd93).

## v3.32.0 - 2023-11-29

- Added ability to exclude some files from `sources:` by using `exclude:` (#225,
  #1324 by @pd93 and @andreynering).
- The
  [Remote Taskfiles experiment](https://taskfile.dev/experiments/remote-taskfiles)
  now prefers remote files over cached ones by default (#1317, #1345 by @pd93).
- Added `--timeout` flag to the
  [Remote Taskfiles experiment](https://taskfile.dev/experiments/remote-taskfiles)
  (#1317, #1345 by @pd93).
- Fix bug where dynamic `vars:` and `env:` were being executed when they should
  actually be skipped by `platforms:` (#1273, #1377 by @andreynering).
- Fix `schema.json` to make `silent` valid in `cmds` that use `for` (#1385,
  #1386 by @iainvm).
- Add new `--no-status` flag to skip expensive status checks when running
  `task --list --json` (#1348, #1368 by @amancevice).

## v3.31.0 - 2023-10-07

- Enabled the `--yes` flag for the
  [Remote Taskfiles experiment](https://taskfile.dev/experiments/remote-taskfiles)
  (#1317, #1344 by @pd93).
- Add ability to set `watch: true` in a task to automatically run it in watch
  mode (#231, #1361 by @andreynering).
- Fixed a bug on the watch mode where paths that contained `.git` (like
  `.github`), for example, were also being ignored (#1356 by @butuzov).
- Fixed a nil pointer error when running a Taskfile with no contents (#1341,
  #1342 by @pd93).
- Added a new [exit code](https://taskfile.dev/api/#exit-codes) (107) for when a
  Taskfile does not contain a schema version (#1342 by @pd93).
- Increased limit of maximum task calls from 100 to 1000 for now, as some people
  have been reaching this limit organically now that we have loops. This check
  exists to detect recursive calls, but will be removed in favor of a better
  algorithm soon (#1321, #1332).
- Fixed templating on descriptions on `task --list` (#1343 by @blackjid).
- Fixed a bug where precondition errors were incorrectly being printed when task
  execution was aborted (#1337, #1338 by @sylv-io).

## v3.30.1 - 2023-09-14

- Fixed a regression where some special variables weren't being set correctly
  (#1331, #1334 by @pd93).

## v3.30.0 - 2023-09-13

- Prep work for Remote Taskfiles (#1316 by @pd93).
- Added the
  [Remote Taskfiles experiment](https://taskfile.dev/experiments/remote-taskfiles)
  as a draft (#1152, #1317 by @pd93).
- Improve performance of content checksumming on `sources:` by replacing md5
  with [XXH3](https://xxhash.com/) which is much faster. This is a soft breaking
  change because checksums will be invalidated when upgrading to this release
  (#1325 by @ReillyBrogan).

## v3.29.1 - 2023-08-26

- Update to Go 1.21 (bump minimum version to 1.20) (#1302 by @pd93)
- Fix a missing a line break on log when using `--watch` mode (#1285, #1297 by
  @FilipSolich).
- Fix `defer` on JSON Schema (#1288 by @calvinmclean and @andreynering).
- Fix bug in usage of special variables like `{{.USER_WORKING_DIR}}` in
  combination with `includes` (#1046, #1205, #1250, #1293, #1312, #1274 by
  @andarto, #1309 by @andreynering).
- Fix bug on `--status` flag. Running this flag should not have side-effects: it
  should not update the checksum on `.task`, only report its status (#1305,
  #1307 by @visciang, #1313 by @andreynering).

## v3.28.0 - 2023-07-24

- Added the ability to
  [loop over commands and tasks](https://taskfile.dev/usage/#looping-over-values)
  using `for` (#82, #1220 by @pd93).
- Fixed variable propagation in multi-level includes (#778, #996, #1256 by
  @hudclark).
- Fixed a bug where the `--exit-code` code flag was not returning the correct
  exit code when calling commands indirectly (#1266, #1270 by @pd93).
- Fixed a `nil` panic when a dependency was commented out or left empty (#1263
  by @neomantra).

## v3.27.1 - 2023-06-30

- Fix panic when a `.env` directory (not file) is present on current directory
  (#1244, #1245 by @pd93).

## v3.27.0 - 2023-06-29

- Allow Taskfiles starting with lowercase characters (#947, #1221 by @pd93).
  - e.g. `taskfile.yml`, `taskfile.yaml`, `taskfile.dist.yml` &
    `taskfile.dist.yaml`
- Bug fixes were made to the
  [npm installation method](https://taskfile.dev/installation/#npm). (#1190, by
  @sounisi5011).
- Added the
  [gentle force experiment](https://taskfile.dev/experiments/gentle-force) as a
  draft (#1200, #1216 by @pd93).
- Added an `--experiments` flag to allow you to see which experiments are
  enabled (#1242 by @pd93).
- Added ability to specify which variables are required in a task (#1203, #1204
  by @benc-uk).

## v3.26.0 - 2023-06-10

- Only rewrite checksum files in `.task` if the checksum has changed (#1185,
  #1194 by @deviantintegral).
- Added [experiments documentation](https://taskfile.dev/experiments) to the
  website (#1198 by @pd93).
- Deprecated `version: 2` schema. This will be removed in the next major release
  (#1197, #1198, #1199 by @pd93).
- Added a new `prompt:` prop to set a warning prompt to be shown before running
  a potential dangerous task (#100, #1163 by @MaxCheetham,
  [Documentation](https://taskfile.dev/usage/#warning-prompts)).
- Added support for single command task syntax. With this change, it's now
  possible to declare just `cmd:` in a task, avoiding the more complex
  `cmds: []` when you have only a single command for that task (#1130, #1131 by
  @timdp).

## v3.25.0 - 2023-05-22

- Support `silent:` when calling another tasks (#680, #1142 by @danquah).
- Improve PowerShell completion script (#1168 by @trim21).
- Add more languages to the website menu and show translation progress
  percentage (#1173 by @misitebao).
- Starting on this release, official binaries for FreeBSD will be available to
  download (#1068 by @andreynering).
- Fix some errors being unintendedly suppressed (#1134 by @clintmod).
- Fix a nil pointer error when `version` is omitted from a Taskfile (#1148,
  #1149 by @pd93).
- Fix duplicate error message when a task does not exists (#1141, #1144 by
  @pd93).

## v3.24.0 - 2023-04-15

- Fix Fish shell completion for tasks with aliases (#1113 by @patricksjackson).
- The default branch was renamed from `master` to `main` (#1049, #1048 by
  @pd93).
- Fix bug where "up-to-date" logs were not being omitted for silent tasks (#546,
  #1107 by @danquah).
- Add `.hg` (Mercurial) to the list of ignored directories when using `--watch`
  (#1098 by @misery).
- More improvements to the release tool (#1096 by @pd93).
- Enforce [gofumpt](https://github.com/mvdan/gofumpt) linter (#1099 by @pd93)
- Add `--sort` flag for use with `--list` and `--list-all` (#946, #1105 by
  @pd93).
- Task now has [custom exit codes](https://taskfile.dev/api/#exit-codes)
  depending on the error (#1114 by @pd93).

## v3.23.0 - 2023-03-26

Task now has an
[official extension for Visual Studio Code](https://marketplace.visualstudio.com/items?itemName=task.vscode-task)
contributed by @pd93! :tada: The extension is maintained in a
[new repository](https://github.com/go-task/vscode-task) under the `go-task`
organization. We're looking to gather feedback from the community so please give
it a go and let us know what you think via a
[discussion](https://github.com/go-task/vscode-task/discussions),
[issue](https://github.com/go-task/vscode-task/issues) or on our
[Discord](https://discord.gg/6TY36E39UK)!

> **NOTE:** The extension _requires_ v3.23.0 to be installed in order to work.

- The website was integrated with
  [Crowdin](https://crowdin.com/project/taskfile) to allow the community to
  contribute with translations! [Chinese](https://taskfile.dev/zh-Hans/) is the
  first language available (#1057, #1058 by @misitebao).
- Added task location data to the `--json` flag output (#1056 by @pd93)
- Change the name of the file generated by `task --init` from `Taskfile.yaml` to
  `Taskfile.yml` (#1062 by @misitebao).
- Added new `splitArgs` template function
  (`{{splitArgs "foo bar 'foo bar baz'"}}`) to ensure string is split as
  arguments (#1040, #1059 by @dhanusaputra).
- Fix the value of `{{.CHECKSUM}}` variable in status (#1076, #1080 by @pd93).
- Fixed deep copy implementation (#1072 by @pd93)
- Created a tool to assist with releases (#1086 by @pd93).

## v3.22.0 - 2023-03-10

- Add a brand new `--global` (`-g`) flag that will run a Taskfile from your
  `$HOME` directory. This is useful to have automation that you can run from
  anywhere in your system!
  ([Documentation](https://taskfile.dev/usage/#running-a-global-taskfile), #1029
  by @andreynering).
- Add ability to set `error_only: true` on the `group` output mode. This will
  instruct Task to only print a command output if it returned with a non-zero
  exit code (#664, #1022 by @jaedle).
- Fixed bug where `.task/checksum` file was sometimes not being created when
  task also declares a `status:` (#840, #1035 by @harelwa, #1037 by @pd93).
- Refactored and decoupled fingerprinting from the main Task executor (#1039 by
  @pd93).
- Fixed deadlock issue when using `run: once` (#715, #1025 by
  @theunrepentantgeek).

## v3.21.0 - 2023-02-22

- Added new `TASK_VERSION` special variable (#990, #1014 by @ja1code).
- Fixed a bug where tasks were sometimes incorrectly marked as internal (#1007
  by @pd93).
- Update to Go 1.20 (bump minimum version to 1.19) (#1010 by @pd93)
- Added environment variable `FORCE_COLOR` support to force color output. Useful
  for environments without TTY (#1003 by @automation-stack)

## v3.20.0 - 2023-01-14

- Improve behavior and performance of status checking when using the `timestamp`
  mode (#976, #977 by @aminya).
- Performance optimizations were made for large Taskfiles (#982 by @pd93).
- Add ability to configure options for the
  [`set`](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html)
  and
  [`shopt`](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html)
  builtins (#908, #929 by @pd93,
  [Documentation](http://taskfile.dev/usage/#set-and-shopt)).
- Add new `platforms:` attribute to `task` and `cmd`, so it's now possible to
  choose in which platforms that given task or command will be run on. Possible
  values are operating system (GOOS), architecture (GOARCH) or a combination of
  the two. Example: `platforms: [linux]`, `platforms: [amd64]` or
  `platforms: [linux/amd64]`. Other platforms will be skipped (#978, #980 by
  @leaanthony).

## v3.19.1 - 2022-12-31

- Small bug fix: closing `Taskfile.yml` once we're done reading it (#963, #964
  by @HeCorr).
- Fixes a bug in v2 that caused a panic when using a `Taskfile_{{OS}}.yml` file
  (#961, #971 by @pd93).
- Fixed a bug where watch intervals set in the Taskfile were not being respected
  (#969, #970 by @pd93)
- Add `--json` flag (alias `-j`) with the intent to improve support for code
  editors and add room to other possible integrations. This is basic for now,
  but we plan to add more info in the near future (#936 by @davidalpert, #764).

## v3.19.0 - 2022-12-05

- Installation via npm now supports [pnpm](https://pnpm.io/) as well
  ([go-task/go-npm#2](https://github.com/go-task/go-npm/issues/2),
  [go-task/go-npm#3](https://github.com/go-task/go-npm/pull/3)).
- It's now possible to run Taskfiles from subdirectories! A new
  `USER_WORKING_DIR` special variable was added to add even more flexibility for
  monorepos (#289, #920).
- Add task-level `dotenv` support (#389, #904).
- It's now possible to use global level variables on `includes` (#942, #943).
- The website got a brand new
  [translation to Chinese](https://task-zh.readthedocs.io/zh_CN/latest/) by
  [@DeronW](https://github.com/DeronW). Thanks!

## v3.18.0 - 2022-11-12

- Show aliases on `task --list --silent` (`task --ls`). This means that aliases
  will be completed by the completion scripts (#919).
- Tasks in the root Taskfile will now be displayed first in
  `--list`/`--list-all` output (#806, #890).
- It's now possible to call a `default` task in an included Taskfile by using
  just the namespace. For example: `docs:default` is now automatically aliased
  to `docs` (#661, #815).

## v3.17.0 - 2022-10-14

- Add a "Did you mean ...?" suggestion when a task does not exits another one
  with a similar name is found (#867, #880).
- Now YAML parse errors will print which Taskfile failed to parse (#885, #887).
- Add ability to set `aliases` for tasks and namespaces (#268, #340, #879).
- Improvements to Fish shell completion (#897).
- Added ability to set a different watch interval by setting `interval: '500ms'`
  or using the `--interval=500ms` flag (#813, #865).
- Add colored output to `--list`, `--list-all` and `--summary` flags (#845,
  #874).
- Fix unexpected behavior where `label:` was being shown instead of the task
  name on `--list` (#603, #877).

## v3.16.0 - 2022-09-29

- Add `npm` as new installation method: `npm i -g @go-task/cli` (#870, #871,
  [npm package](https://www.npmjs.com/package/@go-task/cli)).
- Add support to marking tasks and includes as internal, which will hide them
  from `--list` and `--list-all` (#818).

## v3.15.2 - 2022-09-08

- Fix error when using variable in `env:` introduced in the previous release
  (#858, #866).
- Fix handling of `CLI_ARGS` (`--`) in Bash completion (#863).
- On zsh completion, add ability to replace `--list-all` with `--list` as
  already possible on the Bash completion (#861).

## v3.15.0 - 2022-09-03

- Add new special variables `ROOT_DIR` and `TASKFILE_DIR`. This was a highly
  requested feature (#215, #857,
  [Documentation](https://taskfile.dev/api/#special-variables)).
- Follow symlinks on `sources` (#826, #831).
- Improvements and fixes to Bash completion (#835, #844).

## v3.14.1 - 2022-08-03

- Always resolve relative include paths relative to the including Taskfile
  (#822, #823).
- Fix ZSH and PowerShell completions to consider all tasks instead of just the
  public ones (those with descriptions) (#803).

## v3.14.0 - 2022-07-08

- Add ability to override the `.task` directory location with the
  `TASK_TEMP_DIR` environment variable.
- Allow to override Task colors using environment variables: `TASK_COLOR_RESET`,
  `TASK_COLOR_BLUE`, `TASK_COLOR_GREEN`, `TASK_COLOR_CYAN`, `TASK_COLOR_YELLOW`,
  `TASK_COLOR_MAGENTA` and `TASK_COLOR_RED` (#568, #792).
- Fixed bug when using the `output: group` mode where STDOUT and STDERR were
  being print in separated blocks instead of in the right order (#779).
- Starting on this release, ARM architecture binaries are been released to Snap
  as well (#795).
- i386 binaries won't be available anymore on Snap because Ubuntu removed the
  support for this architecture.
- Upgrade mvdan.cc/sh, which fixes a bug with associative arrays (#785,
  [mvdan/sh#884](https://github.com/mvdan/sh/issues/884),
  [mvdan/sh#893](https://github.com/mvdan/sh/pull/893)).

## v3.13.0 - 2022-06-13

- Added `-n` as an alias to `--dry` (#776, #777).
- Fix behavior of interrupt (SIGINT, SIGTERM) signals. Task will now give time
  for the processes running to do cleanup work (#458, #479, #728, #769).
- Add new `--exit-code` (`-x`) flag that will pass-through the exit form the
  command being ran (#755).

## v3.12.1 - 2022-05-10

- Fixed bug where, on Windows, variables were ending with `\r` because we were
  only removing the final `\n` but not `\r\n` (#717).

## v3.12.0 - 2022-03-31

- The `--list` and `--list-all` flags can now be combined with the `--silent`
  flag to print the task names only, without their description (#691).
- Added support for multi-level inclusion of Taskfiles. This means that included
  Taskfiles can also include other Taskfiles. Before this was limited to one
  level (#390, #623, #656).
- Add ability to specify vars when including a Taskfile.
  [Check out the documentation](https://taskfile.dev/#/usage?id=vars-of-included-taskfiles)
  for more information (#677).

## v3.11.0 - 2022-02-19

- Task now supports printing begin and end messages when using the `group`
  output mode, useful for grouping tasks in CI systems.
  [Check out the documentation](http://taskfile.dev/#/usage?id=output-syntax)
  for more information (#647, #651).
- Add `Taskfile.dist.yml` and `Taskfile.dist.yaml` to the supported file name
  list.
  [Check out the documentation](https://taskfile.dev/#/usage?id=supported-file-names)
  for more information (#498, #666).

## v3.10.0 - 2022-01-04

- A new `--list-all` (alias `-a`) flag is now available. It's similar to the
  exiting `--list` (`-l`) but prints all tasks, even those without a description
  (#383, #401).
- It's now possible to schedule cleanup commands to run once a task finishes
  with the `defer:` keyword
  ([Documentation](https://taskfile.dev/#/usage?id=doing-task-cleanup-with-defer),
  #475, #626).
- Remove long deprecated and undocumented `$` variable prefix and `^` command
  prefix (#642, #644, #645).
- Add support for `.yaml` extension (as an alternative to `.yml`). This was
  requested multiple times throughout the years. Enjoy! (#183, #184, #369, #584,
  #621).
- Fixed error when computing a variable when the task directory do not exist yet
  (#481, #579).

## v3.9.2 - 2021-12-02

- Upgrade [mvdan/sh](https://github.com/mvdan/sh) which contains a fix a for a
  important regression on Windows (#619,
  [mvdan/sh#768](https://github.com/mvdan/sh/issues/768),
  [mvdan/sh#769](https://github.com/mvdan/sh/pull/769)).

## v3.9.1 - 2021-11-28

- Add logging in verbose mode for when a task starts and finishes (#533, #588).
- Fix an issue with preconditions and context errors (#597, #598).
- Quote each `{{.CLI_ARGS}}` argument to prevent one with spaces to become many
  (#613).
- Fix nil pointer when `cmd:` was left empty (#612, #614).
- Upgrade [mvdan/sh](https://github.com/mvdan/sh) which contains two relevant
  fixes:
  - Fix quote of empty strings in `shellQuote` (#609,
    [mvdan/sh#763](https://github.com/mvdan/sh/issues/763)).
  - Fix issue of wrong environment variable being picked when there's another
    very similar one (#586,
    [mvdan/sh#745](https://github.com/mvdan/sh/pull/745)).
- Install shell completions automatically when installing via Homebrew (#264,
  #592,
  [go-task/homebrew-tap#2](https://github.com/go-task/homebrew-tap/pull/2)).

## v3.9.0 - 2021-10-02

- A new `shellQuote` function was added to the template system
  (`{{shellQuote "a string"}}`) to ensure a string is safe for use in shell
  ([mvdan/sh#727](https://github.com/mvdan/sh/pull/727),
  [mvdan/sh#737](https://github.com/mvdan/sh/pull/737),
  [Documentation](https://pkg.go.dev/mvdan.cc/sh/v3@v3.4.0/syntax#Quote))
- In this version [mvdan.cc/sh](https://github.com/mvdan/sh) was upgraded with
  some small fixes and features
  - The `read -p` flag is now supported (#314,
    [mvdan/sh#551](https://github.com/mvdan/sh/issues/551),
    [mvdan/sh#772](https://github.com/mvdan/sh/pull/722))
  - The `pwd -P` and `pwd -L` flags are now supported (#553,
    [mvdan/sh#724](https://github.com/mvdan/sh/issues/724),
    [mvdan/sh#728](https://github.com/mvdan/sh/pull/728))
  - The `$GID` environment variable is now correctly being set (#561,
    [mvdan/sh#723](https://github.com/mvdan/sh/pull/723))

## v3.8.0 - 2021-09-26

- Add `interactive: true` setting to improve support for interactive CLI apps
  (#217, #563).
- Fix some `nil` errors (#534, #573).
- Add ability to declare an included Taskfile as optional (#519, #552).
- Add support for including Taskfiles in the home directory by using `~` (#539,
  #557).

## v3.7.3 - 2021-09-04

- Add official support to Apple M1 (#564, #567).
- Our [official Homebrew tap](https://github.com/go-task/homebrew-tap) will
  support more platforms, including Apple M1

## v3.7.0 - 2021-07-31

- Add `run:` setting to control if tasks should run multiple times or not.
  Available options are `always` (the default), `when_changed` (if a variable
  modified the task) and `once` (run only once no matter what). This is a long
  time requested feature. Enjoy! (#53, #359).

## v3.6.0 - 2021-07-10

- Allow using both `sources:` and `status:` in the same task (#411, #427, #477).
- Small optimization and bug fix: don't compute variables if not needed for
  `dotenv:` (#517).

## v3.5.0 - 2021-07-04

- Add support for interpolation in `dotenv:` (#433, #434, #453).

## v3.4.3 - 2021-05-30

- Add support for the `NO_COLOR` environment variable. (#459,
  [fatih/color#137](https://github.com/fatih/color/pull/137)).
- Fix bug where sources were not considering the right directory in `--watch`
  mode (#484, #485).

## v3.4.2 - 2021-04-23

- On watch, report which file failed to read (#472).
- Do not try to catch SIGKILL signal, which are not actually possible (#476).
- Improve version reporting when building Task from source using Go Modules
  (#462, #473).

## v3.4.1 - 2021-04-17

- Improve error reporting when parsing YAML: in some situations where you would
  just see an generic error, you'll now see the actual error with more detail:
  the YAML line the failed to parse, for example (#467).
- A JSON Schema was published [here](https://json.schemastore.org/taskfile.json)
  and is automatically being used by some editors like Visual Studio Code
  (#135).
- Print task name before the command in the log output (#398).

## v3.3.0 - 2021-03-20

- Add support for delegating CLI arguments to commands with `--` and a special
  `CLI_ARGS` variable (#327).
- Add a `--concurrency` (alias `-C`) flag, to limit the number of tasks that run
  concurrently. This is useful for heavy workloads. (#345).

## v3.2.2 - 2021-01-12

- Improve performance of `--list` and `--summary` by skipping running shell
  variables for these flags (#332).
- Fixed a bug where an environment in a Taskfile was not always overridable by
  the system environment (#425).
- Fixed environment from .env files not being available as variables (#379).
- The install script is now working for ARM platforms (#428).

## v3.2.1 - 2021-01-09

- Fixed some bugs and regressions regarding dynamic variables and directories
  (#426).
- The [slim-sprig](https://github.com/go-task/slim-sprig) package was updated
  with the upstream [sprig](https://github.com/Masterminds/sprig).

## v3.2.0 - 2021-01-07

- Fix the `.task` directory being created in the task directory instead of the
  Taskfile directory (#247).
- Fix a bug where dynamic variables (those declared with `sh:`) were not running
  in the task directory when the task has a custom dir or it was in an included
  Taskfile (#384).
- The watch feature (via the `--watch` flag) got a few different bug fixes and
  should be more stable now (#423, #365).

## v3.1.0 - 2021-01-03

- Fix a bug when the checksum up-to-date resolution is used by a task with a
  custom `label:` attribute (#412).
- Starting from this release, we're releasing official ARMv6 and ARM64 binaries
  for Linux (#375, #418).
- Task now respects the order of declaration of included Taskfiles when
  evaluating variables declaring by them (#393).
- `set -e` is now automatically set on every command. This was done to fix an
  issue where multiline string commands wouldn't really fail unless the sentence
  was in the last line (#403).

## v3.0.1 - 2020-12-26

- Allow use as a library by moving the required packages out of the `internal`
  directory (#358).
- Do not error if a specified dotenv file does not exist (#378, #385).
- Fix panic when you have empty tasks in your Taskfile (#338, #362).

## v3.0.0 - 2020-08-16

- On `v3`, all CLI variables will be considered global variables (#336, #341)
- Add support to `.env` like files (#324, #356).
- Add `label:` to task so you can override the task name in the logs (#321,
  #337).
- Refactor how variables work on version 3 (#311).
- Disallow `expansions` on v3 since it has no effect.
- `Taskvars.yml` is not automatically included anymore.
- `Taskfile_{{OS}}.yml` is not automatically included anymore.
- Allow interpolation on `includes`, so you can manually include a Taskfile
  based on operation system, for example.
- Expose `.TASK` variable in templates with the task name (#252).
- Implement short task syntax (#194, #240).
- Added option to make included Taskfile run commands on its own directory
  (#260, #144)
- Taskfiles in version 1 are not supported anymore (#237).
- Added global `method:` option. With this option, you can set a default method
  to all tasks in a Taskfile (#246).
- Changed default method from `timestamp` to `checksum` (#246).
- New magic variables are now available when using `status:`: `.TIMESTAMP` which
  contains the greatest modification date from the files listed in `sources:`,
  and `.CHECKSUM`, which contains a checksum of all files listed in `status:`.
  This is useful for manual checking when using external, or even remote,
  artifacts when using `status:` (#216).
- We're now using [slim-sprig](https://github.com/go-task/slim-sprig) instead of
  [sprig](https://github.com/Masterminds/sprig), which allowed a file size
  reduction of about 22% (#219).
- We now use some colors on Task output to better distinguish message types -
  commands are green, errors are red, etc (#207).

## v2.8.1 - 2020-05-20

- Fix error code for the `--help` flag (#300, #330).
- Print version to stdout instead of stderr (#299, #329).
- Suppress `context` errors when using the `--watch` flag (#313, #317).
- Support templating on description (#276, #283).

## v2.8.0 - 2019-12-07

- Add `--parallel` flag (alias `-p`) to run tasks given by the command line in
  parallel (#266).
- Fixed bug where calling the `task` CLI only informing global vars would not
  execute the `default` task.
- Add ability to silent all tasks by adding `silent: true` a the root of the
  Taskfile.

## v2.7.1 - 2019-11-10

- Fix error being raised when `exit 0` was called (#251).

## v2.7.0 - 2019-09-22

- Fixed panic bug when assigning a global variable (#229, #243).
- A task with `method: checksum` will now re-run if generated files are deleted
  (#228, #238).

## v2.6.0 - 2019-07-21

- Fixed some bugs regarding minor version checks on `version:`.
- Add `preconditions:` to task (#205).
- Create directory informed on `dir:` if it doesn't exist (#209, #211).
- We now have a `--taskfile` flag (alias `-t`), which can be used to run another
  Taskfile (other than the default `Taskfile.yml`) (#221).
- It's now possible to install Task using Homebrew on Linux
  ([go-task/homebrew-tap#1](https://github.com/go-task/homebrew-tap/pull/1)).

## v2.5.2 - 2019-05-11

- Reverted YAML upgrade due issues with CRLF on Windows (#201,
  [go-yaml/yaml#450](https://github.com/go-yaml/yaml/issues/450)).
- Allow setting global variables through the CLI (#192).

## 2.5.1 - 2019-04-27

- Fixed some issues with interactive command line tools, where sometimes the
  output were not being shown, and similar issues (#114, #190, #200).
- Upgraded [go-yaml/yaml](https://github.com/go-yaml/yaml) from v2 to v3.

## v2.5.0 - 2019-03-16

- We moved from the taskfile.org domain to the new fancy taskfile.dev domain.
  While stuff is being redirected, we strongly recommend to everyone that use
  [this install script](https://taskfile.dev/#/installation?id=install-script)
  to use the new taskfile.dev domain on scripts from now on.
- Fixed to the ZSH completion (#182).
- Add
  [`--summary` flag along with `summary:` task attribute](https://taskfile.org/#/usage?id=display-summary-of-task)
  (#180).

## v2.4.0 - 2019-02-21

- Allow calling a task of the root Taskfile from an included Taskfile by
  prefixing it with `:` (#161, #172).
- Add flag to override the `output` option (#173).
- Fix bug where Task was persisting the new checksum on the disk when the Dry
  Mode is enabled (#166).
- Fix file timestamp issue when the file name has spaces (#176).
- Mitigating path expanding issues on Windows (#170).

## v2.3.0 - 2019-01-02

- On Windows, Task can now be installed using [Scoop](https://scoop.sh/) (#152).
- Fixed issue with file/directory globing (#153).
- Added ability to globally set environment variables (#138, #159).

## v2.2.1 - 2018-12-09

- This repository now uses Go Modules (#143). We'll still keep the `vendor`
  directory in sync for some time, though;
- Fixing a bug when the Taskfile has no tasks but includes another Taskfile
  (#150);
- Fix a bug when calling another task or a dependency in an included Taskfile
  (#151).

## v2.2.0 - 2018-10-25

- Added support for
  [including other Taskfiles](https://taskfile.org/#/usage?id=including-other-taskfiles)
  (#98)
  - This should be considered experimental. For now, only including local files
    is supported, but support for including remote Taskfiles is being discussed.
    If you have any feedback, please comment on #98.
- Task now have a dedicated documentation site: https://taskfile.org
  - Thanks to [Docsify](https://docsify.js.org/) for making this pretty easy. To
    check the source code, just take a look at the
    [docs](https://github.com/go-task/task/tree/main/docs) directory of this
    repository. Contributions to the documentation is really appreciated.

## v2.1.1 - 2018-09-17

- Fix suggestion to use `task --init` not being shown anymore (when a
  `Taskfile.yml` is not found)
- Fix error when using checksum method and no file exists for a source glob
  (#131)
- Fix signal handling when the `--watch` flag is given (#132)

## v2.1.0 - 2018-08-19

- Add a `ignore_error` option to task and command (#123)
- Add a dry run mode (`--dry` flag) (#126)

## v2.0.3 - 2018-06-24

- Expand environment variables on "dir", "sources" and "generates" (#116)
- Fix YAML merging syntax (#112)
- Add ZSH completion (#111)
- Implement new `output` option. Please check out the
  [documentation](https://github.com/go-task/task#output-syntax)

## v2.0.2 - 2018-05-01

- Fix merging of YAML anchors (#112)

## v2.0.1 - 2018-03-11

- Fixes panic on `task --list`

## v2.0.0 - 2018-03-08

Version 2.0.0 is here, with a new Taskfile format.

Please, make sure to read the
[Taskfile versions](https://github.com/go-task/task/blob/main/TASKFILE_VERSIONS.md)
document, since it describes in depth what changed for this version.

- New Taskfile version 2 (#77)
- Possibility to have global variables in the `Taskfile.yml` instead of
  `Taskvars.yml` (#66)
- Small improvements and fixes

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
- Added support for multiline variables from sh (#64)
- Fixes env: remove square braces and evaluate shell (#62)
- Watch: change watch library and few fixes and improvements
- When use watching, cancel and restart long running process on file change (#59
  and #60)

## v1.4.2 - 2017-07-30

- Flag to set directory of execution
- Always echo command if is verbose mode
- Add silent mode to disable echoing of commands
- Fixes and improvements of variables (#56)

## v1.4.1 - 2017-07-15

- Allow use of YAML for dynamic variables instead of $ prefix
  - `VAR: {sh: echo Hello}` instead of `VAR: $echo Hello`
- Add `--list` (or `-l`) flag to print existing tasks
- OS specific Taskvars file (e.g. `Taskvars_windows.yml`, `Taskvars_linux.yml`,
  etc)
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
  - Now, `cmd` is not used anymore on Windows. Always use Bash-like syntax for
    your commands, even on Windows.
- Add "ToSlash" and "FromSlash" to template functions
- Use functions defined on github.com/Masterminds/sprig
- Do not redirect stdin while running variables commands
- Using `context` and `errgroup` packages (this will make other tasks to be
  cancelled, if one returned an error)

## v1.2.0 - 2017-04-02

- More tests and Travis integration
- Watch a task (experimental)
- Possibility to call another task
- Fix "=" not being recognized in variables/environment variables
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
