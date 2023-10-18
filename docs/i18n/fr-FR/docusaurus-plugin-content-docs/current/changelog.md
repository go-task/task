---
slug: /changelog/
sidebar_position: 14
---

# Changelog

## v3.31.0 - 2023-10-07

- Enabled the `--yes` flag for the [Remote Taskfiles experiment](https://taskfile.dev/experiments/remote-taskfiles) ([#1344](https://github.com/go-task/task/issues/1344) by [@pd93](https://github.com/pd93)).
- Add ability to set `watch: true` in a task to automatically run it in watch mode ([#231](https://github.com/go-task/task/issues/231), [#1361](https://github.com/go-task/task/issues/1361) by [@andreynering](https://github.com/andreynering)).
- Fixed a bug on the watch mode where paths that contained `.git` (like `.github`), for example, were also being ignored ([#1356](https://github.com/go-task/task/issues/1356) by [@butuzov](https://github.com/butuzov)).
- Fixed a nil pointer error when running a Taskfile with no contents ([#1341](https://github.com/go-task/task/issues/1341), [#1342](https://github.com/go-task/task/issues/1342) by [@pd93](https://github.com/pd93)).
- Added a new [exit code](https://taskfile.dev/api/#exit-codes) (107) for when a Taskfile does not contain a schema version ([#1342](https://github.com/go-task/task/issues/1342) by [@pd93](https://github.com/pd93)).
- Increased limit of maximum task calls from 100 to 1000 for now, as some people have been reaching this limit organically now that we have loops. This check exists to detect recursive calls, but will be removed in favor of a better algorithm soon ([#1321](https://github.com/go-task/task/issues/1321), [#1332](https://github.com/go-task/task/issues/1332)).
- Fixed templating on descriptions on `task --list` ([#1343](https://github.com/go-task/task/issues/1343) by [@blackjid](https://github.com/blackjid)).
- Fixed a bug where precondition errors were incorrectly being printed when task execution was aborted ([#1337](https://github.com/go-task/task/issues/1337), [#1338](https://github.com/go-task/task/issues/1338) by [@sylv](https://github.com/sylv)-io).

## v3.30.1 - 2023-09-14

- Fixed a regression where some special variables weren't being set correctly ([#1331](https://github.com/go-task/task/issues/1331), [#1334](https://github.com/go-task/task/issues/1334) by [@pd93](https://github.com/pd93)).

## v3.30.0 - 2023-09-13

- Prep work for Remote Taskfiles ([#1316](https://github.com/go-task/task/issues/1316) by [@pd93](https://github.com/pd93)).
- Added the [Remote Taskfiles experiment](https://taskfile.dev/experiments/remote-taskfiles) as a draft ([#1152](https://github.com/go-task/task/issues/1152), [#1317](https://github.com/go-task/task/issues/1317) by [@pd93](https://github.com/pd93)).
- Improve performance of content checksuming on `sources:` by replacing md5 with [XXH3](https://xxhash.com/) which is much faster. This is a soft breaking change because checksums will be invalidated when upgrading to this release ([#1325](https://github.com/go-task/task/issues/1325) by [@ReillyBrogan](https://github.com/ReillyBrogan)).

## v3.29.1 - 2023-08-26

- Update to Go 1.21 (bump minimum version to 1.20) ([#1302](https://github.com/go-task/task/issues/1302) by [@pd93](https://github.com/pd93))
- Fix a missing a line break on log when using `--watch` mode ([#1285](https://github.com/go-task/task/issues/1285), [#1297](https://github.com/go-task/task/issues/1297) by [@FilipSolich](https://github.com/FilipSolich)).
- Fix `defer` on JSON Schema ([#1288](https://github.com/go-task/task/issues/1288) by [@calvinmclean](https://github.com/calvinmclean) and [@andreynering](https://github.com/andreynering)).
- Fix bug in usage of special variables like `{{.USER_WORKING_DIR}}` in combination with `includes` ([#1046](https://github.com/go-task/task/issues/1046), [#1205](https://github.com/go-task/task/issues/1205), [#1250](https://github.com/go-task/task/issues/1250), [#1293](https://github.com/go-task/task/issues/1293), [#1312](https://github.com/go-task/task/issues/1312), [#1274](https://github.com/go-task/task/issues/1274) by [@andarto](https://github.com/andarto), [#1309](https://github.com/go-task/task/issues/1309) by [@andreynering](https://github.com/andreynering)).
- Fix bug on `--status` flag. Running this flag should not have side-effects: it should not update the checksum on `.task`, only report its status ([#1305](https://github.com/go-task/task/issues/1305), [#1307](https://github.com/go-task/task/issues/1307) by [@visciang](https://github.com/visciang), [#1313](https://github.com/go-task/task/issues/1313) by [@andreynering](https://github.com/andreynering)).

## v3.28.0 - 2023-07-24

- Added the ability to [loop over commands and tasks](https://taskfile.dev/usage/#looping-over-values) using `for` ([#82](https://github.com/go-task/task/issues/82), [#1220](https://github.com/go-task/task/issues/1220) by [@pd93](https://github.com/pd93)).
- Fixed variable propagation in multi-level includes ([#778](https://github.com/go-task/task/issues/778), [#996](https://github.com/go-task/task/issues/996), [#1256](https://github.com/go-task/task/issues/1256) by [@hudclark](https://github.com/hudclark)).
- Fixed a bug where the `--exit-code` code flag was not returning the correct exit code when calling commands indirectly ([#1266](https://github.com/go-task/task/issues/1266), [#1270](https://github.com/go-task/task/issues/1270) by [@pd93](https://github.com/pd93)).
- Fixed a `nil` panic when a dependency was commented out or left empty ([#1263](https://github.com/go-task/task/issues/1263) by [@neomantra](https://github.com/neomantra)).

## v3.27.1 - 2023-06-30

- Fix panic when a `.env` directory (not file) is present on current directory ([#1244](https://github.com/go-task/task/issues/1244), [#1245](https://github.com/go-task/task/issues/1245) by [@pd93](https://github.com/pd93)).

## v3.27.0 - 2023-06-29

- Allow Taskfiles starting with lowercase characters ([#947](https://github.com/go-task/task/issues/947), [#1221](https://github.com/go-task/task/issues/1221) by [@pd93](https://github.com/pd93)).
  - e.g. `taskfile.yml`, `taskfile.yaml`, `taskfile.dist.yml` & `taskfile.dist.yaml`
- Bug fixes were made to the [npm installation method](https://taskfile.dev/installation/#npm). ([#1190](https://github.com/go-task/task/issues/1190), by [@sounisi5011](https://github.com/sounisi5011)).
- Added the [gentle force experiment](https://taskfile.dev/experiments/gentle-force) as a draft ([#1200](https://github.com/go-task/task/issues/1200), [#1216](https://github.com/go-task/task/issues/1216) by [@pd93](https://github.com/pd93)).
- Added an `--experiments` flag to allow you to see which experiments are enabled ([#1242](https://github.com/go-task/task/issues/1242) by [@pd93](https://github.com/pd93)).
- Added ability to specify which variables are required in a task ([#1203](https://github.com/go-task/task/issues/1203), [#1204](https://github.com/go-task/task/issues/1204) by [@benc](https://github.com/benc)-uk).

## v3.26.0 - 2023-06-10

- Only rewrite checksum files in `.task` if the checksum has changed ([#1185](https://github.com/go-task/task/issues/1185), [#1194](https://github.com/go-task/task/issues/1194) by [@deviantintegral](https://github.com/deviantintegral)).
- Added [experiments documentation](https://taskfile.dev/experiments) to the website ([#1198](https://github.com/go-task/task/issues/1198) by [@pd93](https://github.com/pd93)).
- Deprecated `version: 2` schema. This will be removed in the next major release ([#1197](https://github.com/go-task/task/issues/1197), [#1198](https://github.com/go-task/task/issues/1198), [#1199](https://github.com/go-task/task/issues/1199) by [@pd93](https://github.com/pd93)).
- Added a new `prompt:` prop to set a warning prompt to be shown before running a potential dangurous task ([#100](https://github.com/go-task/task/issues/100), [#1163](https://github.com/go-task/task/issues/1163) by [@MaxCheetham](https://github.com/MaxCheetham), [Documentation](https://taskfile.dev/usage/#warning-prompts)).
- Added support for single command task syntax. With this change, it's now possible to declare just `cmd:` in a task, avoiding the more complex `cmds: []` when you have only a single command for that task ([#1130](https://github.com/go-task/task/issues/1130), [#1131](https://github.com/go-task/task/issues/1131) by [@timdp](https://github.com/timdp)).

## v3.25.0 - 2023-05-22

- Support `silent:` when calling another tasks ([#680](https://github.com/go-task/task/issues/680), [#1142](https://github.com/go-task/task/issues/1142) by [@danquah](https://github.com/danquah)).
- Improve PowerShell completion script ([#1168](https://github.com/go-task/task/issues/1168) by [@trim21](https://github.com/trim21)).
- Add more languages to the website menu and show translation progress percentage ([#1173](https://github.com/go-task/task/issues/1173) by [@misitebao](https://github.com/misitebao)).
- Starting on this release, official binaries for FreeBSD will be available to download ([#1068](https://github.com/go-task/task/issues/1068) by [@andreynering](https://github.com/andreynering)).
- Fix some errors being unintendedly supressed ([#1134](https://github.com/go-task/task/issues/1134) by [@clintmod](https://github.com/clintmod)).
- Fix a nil pointer error when `version` is omitted from a Taskfile ([#1148](https://github.com/go-task/task/issues/1148), [#1149](https://github.com/go-task/task/issues/1149) by [@pd93](https://github.com/pd93)).
- Fix duplicate error message when a task does not exists ([#1141](https://github.com/go-task/task/issues/1141), [#1144](https://github.com/go-task/task/issues/1144) by [@pd93](https://github.com/pd93)).

## v3.24.0 - 2023-04-15

- Fix Fish shell completion for tasks with aliases ([#1113](https://github.com/go-task/task/issues/1113) by [@patricksjackson](https://github.com/patricksjackson)).
- The default branch was renamed from `master` to `main` ([#1049](https://github.com/go-task/task/issues/1049), [#1048](https://github.com/go-task/task/issues/1048) by [@pd93](https://github.com/pd93)).
- Fix bug where "up-to-date" logs were not being omitted for silent tasks ([#546](https://github.com/go-task/task/issues/546), [#1107](https://github.com/go-task/task/issues/1107) by [@danquah](https://github.com/danquah)).
- Add `.hg` (Mercurial) to the list of ignored directories when using `--watch` ([#1098](https://github.com/go-task/task/issues/1098) by [@misery](https://github.com/misery)).
- More improvements to the release tool ([#1096](https://github.com/go-task/task/issues/1096) by [@pd93](https://github.com/pd93)).
- Enforce [gofumpt](https://github.com/mvdan/gofumpt) linter ([#1099](https://github.com/go-task/task/issues/1099) by [@pd93](https://github.com/pd93))
- Add `--sort` flag for use with `--list` and `--list-all` ([#946](https://github.com/go-task/task/issues/946), [#1105](https://github.com/go-task/task/issues/1105) by [@pd93](https://github.com/pd93)).
- Task now has [custom exit codes](https://taskfile.dev/api/#exit-codes) depending on the error ([#1114](https://github.com/go-task/task/issues/1114) by [@pd93](https://github.com/pd93)).

## v3.23.0 - 2023-03-26

Task now has an [official extension for Visual Studio Code](https://marketplace.visualstudio.com/items?itemName=task.vscode-task) contributed by [@pd93](https://github.com/pd93)! :tada: The extension is maintained in a [new repository](https://github.com/go-task/vscode-task) under the `go-task` organization. We're looking to gather feedback from the community so please give it a go and let us know what you think via a [discussion](https://github.com/go-task/vscode-task/discussions), [issue](https://github.com/go-task/vscode-task/issues) or on our [Discord](https://discord.gg/6TY36E39UK)!

> **NOTE:** The extension _requires_ v3.23.0 to be installed in order to work.

- The website was integrated with [Crowdin](https://crowdin.com/project/taskfile) to allow the community to contribute with translations! [Chinese](https://taskfile.dev/zh-Hans/) is the first language available ([#1057](https://github.com/go-task/task/issues/1057), [#1058](https://github.com/go-task/task/issues/1058) by [@misitebao](https://github.com/misitebao)).
- Added task location data to the `--json` flag output ([#1056](https://github.com/go-task/task/issues/1056) by [@pd93](https://github.com/pd93))
- Change the name of the file generated by `task --init` from `Taskfile.yaml` to `Taskfile.yml` ([#1062](https://github.com/go-task/task/issues/1062) by [@misitebao](https://github.com/misitebao)).
- Added new `splitArgs` template function (`{{splitArgs "foo bar 'foo bar baz'"}}`) to ensure string is split as arguments ([#1040](https://github.com/go-task/task/issues/1040), [#1059](https://github.com/go-task/task/issues/1059) by [@dhanusaputra](https://github.com/dhanusaputra)).
- Fix the value of `{{.CHECKSUM}}` variable in status ([#1076](https://github.com/go-task/task/issues/1076), [#1080](https://github.com/go-task/task/issues/1080) by [@pd93](https://github.com/pd93)).
- Fixed deep copy implementation ([#1072](https://github.com/go-task/task/issues/1072) by [@pd93](https://github.com/pd93))
- Created a tool to assist with releases ([#1086](https://github.com/go-task/task/issues/1086) by [@pd93](https://github.com/pd93)).

## v3.22.0 - 2023-03-10

- Add a brand new `--global` (`-g`) flag that will run a Taskfile from your `$HOME` directory. This is useful to have automation that you can run from anywhere in your system! ([Documentation](https://taskfile.dev/usage/#running-a-global-taskfile), [#1029](https://github.com/go-task/task/issues/1029) by [@andreynering](https://github.com/andreynering)).
- Add ability to set `error_only: true` on the `group` output mode. This will instruct Task to only print a command output if it returned with a non-zero exit code ([#664](https://github.com/go-task/task/issues/664), [#1022](https://github.com/go-task/task/issues/1022) by [@jaedle](https://github.com/jaedle)).
- Fixed bug where `.task/checksum` file was sometimes not being created when task also declares a `status:` ([#840](https://github.com/go-task/task/issues/840), [#1035](https://github.com/go-task/task/issues/1035) by [@harelwa](https://github.com/harelwa), [#1037](https://github.com/go-task/task/issues/1037) by [@pd93](https://github.com/pd93)).
- Refactored and decoupled fingerprinting from the main Task executor ([#1039](https://github.com/go-task/task/issues/1039) by [@pd93](https://github.com/pd93)).
- Fixed deadlock issue when using `run: once` ([#715](https://github.com/go-task/task/issues/715), [#1025](https://github.com/go-task/task/issues/1025) by [@theunrepentantgeek](https://github.com/theunrepentantgeek)).

## v3.21.0 - 2023-02-22

- Added new `TASK_VERSION` special variable ([#990](https://github.com/go-task/task/issues/990), [#1014](https://github.com/go-task/task/issues/1014) by [@ja1code](https://github.com/ja1code)).
- Fixed a bug where tasks were sometimes incorrectly marked as internal ([#1007](https://github.com/go-task/task/issues/1007) by [@pd93](https://github.com/pd93)).
- Update to Go 1.20 (bump minimum version to 1.19) ([#1010](https://github.com/go-task/task/issues/1010) by [@pd93](https://github.com/pd93))
- Added environment variable `FORCE_COLOR` support to force color output. Usefull for environments without TTY ([#1003](https://github.com/go-task/task/issues/1003) by [@automation](https://github.com/automation)-stack)

## v3.20.0 - 2023-01-14

- Improve behavior and performance of status checking when using the `timestamp` mode ([#976](https://github.com/go-task/task/issues/976), [#977](https://github.com/go-task/task/issues/977) by [@aminya](https://github.com/aminya)).
- Performance optimizations were made for large Taskfiles ([#982](https://github.com/go-task/task/issues/982) by [@pd93](https://github.com/pd93)).
- Add ability to configure options for the [`set`](https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html) and [`shopt`](https://www.gnu.org/software/bash/manual/html_node/The-Shopt-Builtin.html) builtins ([#908](https://github.com/go-task/task/issues/908), [#929](https://github.com/go-task/task/issues/929) by [@pd93](https://github.com/pd93), [Documentation](http://taskfile.dev/usage/#set-and-shopt)).
- Add new `platforms:` attribute to `task` and `cmd`, so it's now possible to choose in which platforms that given task or command will be run on. Possible values are operating system (GOOS), architecture (GOARCH) or a combination of the two. Example: `platforms: [linux]`, `platforms: [amd64]` or `platforms: [linux/amd64]`. Other platforms will be skipped ([#978](https://github.com/go-task/task/issues/978), [#980](https://github.com/go-task/task/issues/980) by [@leaanthony](https://github.com/leaanthony)).

## v3.19.1 - 2022-12-31

- Small bug fix: closing `Taskfile.yml` once we're done reading it ([#963](https://github.com/go-task/task/issues/963), [#964](https://github.com/go-task/task/issues/964) by [@HeCorr](https://github.com/HeCorr)).
- Fixes a bug in v2 that caused a panic when using a `Taskfile_{{OS}}.yml` file ([#961](https://github.com/go-task/task/issues/961), [#971](https://github.com/go-task/task/issues/971) by [@pd93](https://github.com/pd93)).
- Fixed a bug where watch intervals set in the Taskfile were not being respected ([#969](https://github.com/go-task/task/issues/969), [#970](https://github.com/go-task/task/issues/970) by [@pd93](https://github.com/pd93))
- Add `--json` flag (alias `-j`) with the intent to improve support for code editors and add room to other possible integrations. This is basic for now, but we plan to add more info in the near future ([#936](https://github.com/go-task/task/issues/936) by [@davidalpert](https://github.com/davidalpert), [#764](https://github.com/go-task/task/issues/764)).

## v3.19.0 - 2022-12-05

- Installation via npm now supports [pnpm](https://pnpm.io/) as well ([go-task/go-npm\[#2\](https://github.com/go-task/task/issues/2)](https://github.com/go-task/go-npm/issues/2), [go-task/go-npm\[#3\](https://github.com/go-task/task/issues/3)](https://github.com/go-task/go-npm/pull/3)).
- It's now possible to run Taskfiles from subdirectories! A new `USER_WORKING_DIR` special variable was added to add even more flexibility for monorepos ([#289](https://github.com/go-task/task/issues/289), [#920](https://github.com/go-task/task/issues/920)).
- Add task-level `dotenv` support ([#389](https://github.com/go-task/task/issues/389), [#904](https://github.com/go-task/task/issues/904)).
- It's now possible to use global level variables on `includes` ([#942](https://github.com/go-task/task/issues/942), [#943](https://github.com/go-task/task/issues/943)).
- The website got a brand new [translation to Chinese](https://task-zh.readthedocs.io/zh_CN/latest/) by [\[@DeronW\](https://github.com/DeronW)](https://github.com/DeronW). Thanks!

## v3.18.0 - 2022-11-12

- Show aliases on `task --list --silent` (`task --ls`). This means that aliases will be completed by the completion scripts ([#919](https://github.com/go-task/task/issues/919)).
- Tasks in the root Taskfile will now be displayed first in `--list`/`--list-all` output ([#806](https://github.com/go-task/task/issues/806), [#890](https://github.com/go-task/task/issues/890)).
- It's now possible to call a `default` task in an included Taskfile by using just the namespace. For example: `docs:default` is now automatically aliased to `docs` ([#661](https://github.com/go-task/task/issues/661), [#815](https://github.com/go-task/task/issues/815)).

## v3.17.0 - 2022-10-14

- Add a "Did you mean ...?" suggestion when a task does not exits another one with a similar name is found ([#867](https://github.com/go-task/task/issues/867), [#880](https://github.com/go-task/task/issues/880)).
- Now YAML parse errors will print which Taskfile failed to parse ([#885](https://github.com/go-task/task/issues/885), [#887](https://github.com/go-task/task/issues/887)).
- Add ability to set `aliases` for tasks and namespaces ([#268](https://github.com/go-task/task/issues/268), [#340](https://github.com/go-task/task/issues/340), [#879](https://github.com/go-task/task/issues/879)).
- Improvements to Fish shell completion ([#897](https://github.com/go-task/task/issues/897)).
- Added ability to set a different watch interval by setting `interval: '500ms'` or using the `--interval=500ms` flag ([#813](https://github.com/go-task/task/issues/813), [#865](https://github.com/go-task/task/issues/865)).
- Add colored output to `--list`, `--list-all` and `--summary` flags ([#845](https://github.com/go-task/task/issues/845), [#874](https://github.com/go-task/task/issues/874)).
- Fix unexpected behavior where `label:` was being shown instead of the task name on `--list` ([#603](https://github.com/go-task/task/issues/603), [#877](https://github.com/go-task/task/issues/877)).

## v3.16.0 - 2022-09-29

- Add `npm` as new installation method: `npm i -g [@go](https://github.com/go)-task/cli` ([#870](https://github.com/go-task/task/issues/870), [#871](https://github.com/go-task/task/issues/871), [npm package](https://www.npmjs.com/package/[@go](https://github.com/go)-task/cli)).
- Add support to marking tasks and includes as internal, which will hide them from `--list` and `--list-all` ([#818](https://github.com/go-task/task/issues/818)).

## v3.15.2 - 2022-09-08

- Fix error when using variable in `env:` introduced in the previous release ([#858](https://github.com/go-task/task/issues/858), [#866](https://github.com/go-task/task/issues/866)).
- Fix handling of `CLI_ARGS` (`--`) in Bash completion ([#863](https://github.com/go-task/task/issues/863)).
- On zsh completion, add ability to replace `--list-all` with `--list` as already possible on the Bash completion ([#861](https://github.com/go-task/task/issues/861)).

## v3.15.0 - 2022-09-03

- Add new special variables `ROOT_DIR` and `TASKFILE_DIR`. This was a highly requested feature ([#215](https://github.com/go-task/task/issues/215), [#857](https://github.com/go-task/task/issues/857), [Documentation](https://taskfile.dev/api/#special-variables)).
- Follow symlinks on `sources` ([#826](https://github.com/go-task/task/issues/826), [#831](https://github.com/go-task/task/issues/831)).
- Improvements and fixes to Bash completion ([#835](https://github.com/go-task/task/issues/835), [#844](https://github.com/go-task/task/issues/844)).

## v3.14.1 - 2022-08-03

- Always resolve relative include paths relative to the including Taskfile ([#822](https://github.com/go-task/task/issues/822), [#823](https://github.com/go-task/task/issues/823)).
- Fix ZSH and PowerShell completions to consider all tasks instead of just the public ones (those with descriptions) ([#803](https://github.com/go-task/task/issues/803)).

## v3.14.0 - 2022-07-08

- Add ability to override the `.task` directory location with the `TASK_TEMP_DIR` environment variable.
- Allow to override Task colors using environment variables: `TASK_COLOR_RESET`, `TASK_COLOR_BLUE`, `TASK_COLOR_GREEN`, `TASK_COLOR_CYAN`, `TASK_COLOR_YELLOW`, `TASK_COLOR_MAGENTA` and `TASK_COLOR_RED` ([#568](https://github.com/go-task/task/issues/568), [#792](https://github.com/go-task/task/issues/792)).
- Fixed bug when using the `output: group` mode where STDOUT and STDERR were being print in separated blocks instead of in the right order ([#779](https://github.com/go-task/task/issues/779)).
- Starting on this release, ARM architecture binaries are been released to Snap as well ([#795](https://github.com/go-task/task/issues/795)).
- i386 binaries won't be available anymore on Snap because Ubuntu removed the support for this architecture.
- Upgrade mvdan.cc/sh, which fixes a bug with associative arrays ([#785](https://github.com/go-task/task/issues/785), [mvdan/sh\[#884\](https://github.com/go-task/task/issues/884)](https://github.com/mvdan/sh/issues/884), [mvdan/sh\[#893\](https://github.com/go-task/task/issues/893)](https://github.com/mvdan/sh/pull/893)).

## v3.13.0 - 2022-06-13

- Added `-n` as an alias to `--dry` ([#776](https://github.com/go-task/task/issues/776), [#777](https://github.com/go-task/task/issues/777)).
- Fix behavior of interrupt (SIGINT, SIGTERM) signals. Task will now give time for the processes running to do cleanup work ([#458](https://github.com/go-task/task/issues/458), [#479](https://github.com/go-task/task/issues/479), [#728](https://github.com/go-task/task/issues/728), [#769](https://github.com/go-task/task/issues/769)).
- Add new `--exit-code` (`-x`) flag that will pass-through the exit form the command being ran ([#755](https://github.com/go-task/task/issues/755)).

## v3.12.1 - 2022-05-10

- Fixed bug where, on Windows, variables were ending with `\r` because we were only removing the final `\n` but not `\r\n` ([#717](https://github.com/go-task/task/issues/717)).

## v3.12.0 - 2022-03-31

- The `--list` and `--list-all` flags can now be combined with the `--silent` flag to print the task names only, without their description ([#691](https://github.com/go-task/task/issues/691)).
- Added support for multi-level inclusion of Taskfiles. This means that included Taskfiles can also include other Taskfiles. Before this was limited to one level ([#390](https://github.com/go-task/task/issues/390), [#623](https://github.com/go-task/task/issues/623), [#656](https://github.com/go-task/task/issues/656)).
- Add ability to specify vars when including a Taskfile. [Check out the documentation](https://taskfile.dev/#/usage?id=vars-of-included-taskfiles) for more information ([#677](https://github.com/go-task/task/issues/677)).

## v3.11.0 - 2022-02-19

- Task now supports printing begin and end messages when using the `group` output mode, useful for grouping tasks in CI systems. [Check out the documentation](http://taskfile.dev/#/usage?id=output-syntax) for more information ([#647](https://github.com/go-task/task/issues/647), [#651](https://github.com/go-task/task/issues/651)).
- Add `Taskfile.dist.yml` and `Taskfile.dist.yaml` to the supported file name list. [Check out the documentation](https://taskfile.dev/#/usage?id=supported-file-names) for more information ([#498](https://github.com/go-task/task/issues/498), [#666](https://github.com/go-task/task/issues/666)).

## v3.10.0 - 2022-01-04

- A new `--list-all` (alias `-a`) flag is now available. It's similar to the exiting `--list` (`-l`) but prints all tasks, even those without a description ([#383](https://github.com/go-task/task/issues/383), [#401](https://github.com/go-task/task/issues/401)).
- It's now possible to schedule cleanup commands to run once a task finishes with the `defer:` keyword ([Documentation](https://taskfile.dev/#/usage?id=doing-task-cleanup-with-defer), [#475](https://github.com/go-task/task/issues/475), [#626](https://github.com/go-task/task/issues/626)).
- Remove long deprecated and undocumented `$` variable prefix and `^` command prefix ([#642](https://github.com/go-task/task/issues/642), [#644](https://github.com/go-task/task/issues/644), [#645](https://github.com/go-task/task/issues/645)).
- Add support for `.yaml` extension (as an alternative to `.yml`). This was requested multiple times throughout the years. Enjoy! ([#183](https://github.com/go-task/task/issues/183), [#184](https://github.com/go-task/task/issues/184), [#369](https://github.com/go-task/task/issues/369), [#584](https://github.com/go-task/task/issues/584), [#621](https://github.com/go-task/task/issues/621)).
- Fixed error when computing a variable when the task directory do not exist yet ([#481](https://github.com/go-task/task/issues/481), [#579](https://github.com/go-task/task/issues/579)).

## v3.9.2 - 2021-12-02

- Upgrade [mvdan/sh](https://github.com/mvdan/sh) which contains a fix a for a important regression on Windows ([#619](https://github.com/go-task/task/issues/619), [mvdan/sh\[#768\](https://github.com/go-task/task/issues/768)](https://github.com/mvdan/sh/issues/768), [mvdan/sh\[#769\](https://github.com/go-task/task/issues/769)](https://github.com/mvdan/sh/pull/769)).

## v3.9.1 - 2021-11-28

- Add logging in verbose mode for when a task starts and finishes ([#533](https://github.com/go-task/task/issues/533), [#588](https://github.com/go-task/task/issues/588)).
- Fix an issue with preconditions and context errors ([#597](https://github.com/go-task/task/issues/597), [#598](https://github.com/go-task/task/issues/598)).
- Quote each `{{.CLI_ARGS}}` argument to prevent one with spaces to become many ([#613](https://github.com/go-task/task/issues/613)).
- Fix nil pointer when `cmd:` was left empty ([#612](https://github.com/go-task/task/issues/612), [#614](https://github.com/go-task/task/issues/614)).
- Upgrade [mvdan/sh](https://github.com/mvdan/sh) which contains two relevant fixes:
  - Fix quote of empty strings in `shellQuote` ([#609](https://github.com/go-task/task/issues/609), [mvdan/sh\[#763\](https://github.com/go-task/task/issues/763)](https://github.com/mvdan/sh/issues/763)).
  - Fix issue of wrong environment variable being picked when there's another very similar one ([#586](https://github.com/go-task/task/issues/586), [mvdan/sh\[#745\](https://github.com/go-task/task/issues/745)](https://github.com/mvdan/sh/pull/745)).
- Install shell completions automatically when installing via Homebrew ([#264](https://github.com/go-task/task/issues/264), [#592](https://github.com/go-task/task/issues/592), [go-task/homebrew-tap\[#2\](https://github.com/go-task/task/issues/2)](https://github.com/go-task/homebrew-tap/pull/2)).

## v3.9.0 - 2021-10-02

- A new `shellQuote` function was added to the template system (`{{shellQuote "a string"}}`) to ensure a string is safe for use in shell ([mvdan/sh\[#727\](https://github.com/go-task/task/issues/727)](https://github.com/mvdan/sh/pull/727), [mvdan/sh\[#737\](https://github.com/go-task/task/issues/737)](https://github.com/mvdan/sh/pull/737), [Documentation](https://pkg.go.dev/mvdan.cc/sh/v3[@v3](https://github.com/v3).4.0/syntax#Quote))
- In this version [mvdan.cc/sh](https://github.com/mvdan/sh) was upgraded with some small fixes and features
  - The `read -p` flag is now supported ([#314](https://github.com/go-task/task/issues/314), [mvdan/sh\[#551\](https://github.com/go-task/task/issues/551)](https://github.com/mvdan/sh/issues/551), [mvdan/sh\[#772\](https://github.com/go-task/task/issues/772)](https://github.com/mvdan/sh/pull/722))
  - The `pwd -P` and `pwd -L` flags are now supported ([#553](https://github.com/go-task/task/issues/553), [mvdan/sh\[#724\](https://github.com/go-task/task/issues/724)](https://github.com/mvdan/sh/issues/724), [mvdan/sh\[#728\](https://github.com/go-task/task/issues/728)](https://github.com/mvdan/sh/pull/728))
  - The `$GID` environment variable is now correctly being set ([#561](https://github.com/go-task/task/issues/561), [mvdan/sh\[#723\](https://github.com/go-task/task/issues/723)](https://github.com/mvdan/sh/pull/723))

## v3.8.0 - 2021-09-26

- Add `interactive: true` setting to improve support for interactive CLI apps ([#217](https://github.com/go-task/task/issues/217), [#563](https://github.com/go-task/task/issues/563)).
- Fix some `nil` errors ([#534](https://github.com/go-task/task/issues/534), [#573](https://github.com/go-task/task/issues/573)).
- Add ability to declare an included Taskfile as optional ([#519](https://github.com/go-task/task/issues/519), [#552](https://github.com/go-task/task/issues/552)).
- Add support for including Taskfiles in the home directory by using `~` ([#539](https://github.com/go-task/task/issues/539), [#557](https://github.com/go-task/task/issues/557)).

## v3.7.3 - 2021-09-04

- Add official support to Apple M1 ([#564](https://github.com/go-task/task/issues/564), [#567](https://github.com/go-task/task/issues/567)).
- Our [official Homebrew tap](https://github.com/go-task/homebrew-tap) will support more platforms, including Apple M1

## v3.7.0 - 2021-07-31

- Add `run:` setting to control if tasks should run multiple times or not. Available options are `always` (the default), `when_changed` (if a variable modified the task) and `once` (run only once no matter what). This is a long time requested feature. Enjoy! ([#53](https://github.com/go-task/task/issues/53), [#359](https://github.com/go-task/task/issues/359)).

## v3.6.0 - 2021-07-10

- Allow using both `sources:` and `status:` in the same task ([#411](https://github.com/go-task/task/issues/411), [#427](https://github.com/go-task/task/issues/427), [#477](https://github.com/go-task/task/issues/477)).
- Small optimization and bug fix: don't compute variables if not needed for `dotenv:` ([#517](https://github.com/go-task/task/issues/517)).

## v3.5.0 - 2021-07-04

- Add support for interpolation in `dotenv:` ([#433](https://github.com/go-task/task/issues/433), [#434](https://github.com/go-task/task/issues/434), [#453](https://github.com/go-task/task/issues/453)).

## v3.4.3 - 2021-05-30

- Add support for the `NO_COLOR` environment variable. ([#459](https://github.com/go-task/task/issues/459), [fatih/color\[#137\](https://github.com/go-task/task/issues/137)](https://github.com/fatih/color/pull/137)).
- Fix bug where sources were not considering the right directory in `--watch` mode ([#484](https://github.com/go-task/task/issues/484), [#485](https://github.com/go-task/task/issues/485)).

## v3.4.2 - 2021-04-23

- On watch, report which file failed to read ([#472](https://github.com/go-task/task/issues/472)).
- Do not try to catch SIGKILL signal, which are not actually possible ([#476](https://github.com/go-task/task/issues/476)).
- Improve version reporting when building Task from source using Go Modules ([#462](https://github.com/go-task/task/issues/462), [#473](https://github.com/go-task/task/issues/473)).

## v3.4.1 - 2021-04-17

- Improve error reporting when parsing YAML: in some situations where you would just see an generic error, you'll now see the actual error with more detail: the YAML line the failed to parse, for example ([#467](https://github.com/go-task/task/issues/467)).
- A JSON Schema was published [here](https://json.schemastore.org/taskfile.json) and is automatically being used by some editors like Visual Studio Code ([#135](https://github.com/go-task/task/issues/135)).
- Print task name before the command in the log output ([#398](https://github.com/go-task/task/issues/398)).

## v3.3.0 - 2021-03-20

- Add support for delegating CLI arguments to commands with `--` and a special `CLI_ARGS` variable ([#327](https://github.com/go-task/task/issues/327)).
- Add a `--concurrency` (alias `-C`) flag, to limit the number of tasks that run concurrently. This is useful for heavy workloads. ([#345](https://github.com/go-task/task/issues/345)).

## v3.2.2 - 2021-01-12

- Improve performance of `--list` and `--summary` by skipping running shell variables for these flags ([#332](https://github.com/go-task/task/issues/332)).
- Fixed a bug where an environment in a Taskfile was not always overridable by the system environment ([#425](https://github.com/go-task/task/issues/425)).
- Fixed environment from .env files not being available as variables ([#379](https://github.com/go-task/task/issues/379)).
- The install script is now working for ARM platforms ([#428](https://github.com/go-task/task/issues/428)).

## v3.2.1 - 2021-01-09

- Fixed some bugs and regressions regarding dynamic variables and directories ([#426](https://github.com/go-task/task/issues/426)).
- The [slim-sprig](https://github.com/go-task/slim-sprig) package was updated with the upstream [sprig](https://github.com/Masterminds/sprig).

## v3.2.0 - 2021-01-07

- Fix the `.task` directory being created in the task directory instead of the Taskfile directory ([#247](https://github.com/go-task/task/issues/247)).
- Fix a bug where dynamic variables (those declared with `sh:`) were not running in the task directory when the task has a custom dir or it was in an included Taskfile ([#384](https://github.com/go-task/task/issues/384)).
- The watch feature (via the `--watch` flag) got a few different bug fixes and should be more stable now ([#423](https://github.com/go-task/task/issues/423), [#365](https://github.com/go-task/task/issues/365)).

## v3.1.0 - 2021-01-03

- Fix a bug when the checksum up-to-date resolution is used by a task with a custom `label:` attribute ([#412](https://github.com/go-task/task/issues/412)).
- Starting from this release, we're releasing official ARMv6 and ARM64 binaries for Linux ([#375](https://github.com/go-task/task/issues/375), [#418](https://github.com/go-task/task/issues/418)).
- Task now respects the order of declaration of included Taskfiles when evaluating variables declaring by them ([#393](https://github.com/go-task/task/issues/393)).
- `set -e` is now automatically set on every command. This was done to fix an issue where multiline string commands wouldn't really fail unless the sentence was in the last line ([#403](https://github.com/go-task/task/issues/403)).

## v3.0.1 - 2020-12-26

- Allow use as a library by moving the required packages out of the `internal` directory ([#358](https://github.com/go-task/task/issues/358)).
- Do not error if a specified dotenv file does not exist ([#378](https://github.com/go-task/task/issues/378), [#385](https://github.com/go-task/task/issues/385)).
- Fix panic when you have empty tasks in your Taskfile ([#338](https://github.com/go-task/task/issues/338), [#362](https://github.com/go-task/task/issues/362)).

## v3.0.0 - 2020-08-16

- On `v3`, all CLI variables will be considered global variables ([#336](https://github.com/go-task/task/issues/336), [#341](https://github.com/go-task/task/issues/341))
- Add support to `.env` like files ([#324](https://github.com/go-task/task/issues/324), [#356](https://github.com/go-task/task/issues/356)).
- Add `label:` to task so you can override the task name in the logs ([\[#321\](https://github.com/go-task/task/issues/321)](https://github.com/go-task/task/issues/321]), [#337](https://github.com/go-task/task/issues/337)).
- Refactor how variables work on version 3 ([#311](https://github.com/go-task/task/issues/311)).
- Disallow `expansions` on v3 since it has no effect.
- `Taskvars.yml` is not automatically included anymore.
- `Taskfile_{{OS}}.yml` is not automatically included anymore.
- Allow interpolation on `includes`, so you can manually include a Taskfile based on operation system, for example.
- Expose `.TASK` variable in templates with the task name ([#252](https://github.com/go-task/task/issues/252)).
- Implement short task syntax ([#194](https://github.com/go-task/task/issues/194), [#240](https://github.com/go-task/task/issues/240)).
- Added option to make included Taskfile run commands on its own directory ([#260](https://github.com/go-task/task/issues/260), [#144](https://github.com/go-task/task/issues/144))
- Taskfiles in version 1 are not supported anymore ([#237](https://github.com/go-task/task/issues/237)).
- Added global `method:` option. With this option, you can set a default method to all tasks in a Taskfile ([#246](https://github.com/go-task/task/issues/246)).
- Changed default method from `timestamp` to `checksum` ([#246](https://github.com/go-task/task/issues/246)).
- New magic variables are now available when using `status:`: `.TIMESTAMP` which contains the greatest modification date from the files listed in `sources:`, and `.CHECKSUM`, which contains a checksum of all files listed in `status:`. This is useful for manual checking when using external, or even remote, artifacts when using `status:` ([#216](https://github.com/go-task/task/issues/216)).
- We're now using [slim-sprig](https://github.com/go-task/slim-sprig) instead of [sprig](https://github.com/Masterminds/sprig), which allowed a file size reduction of about 22% ([#219](https://github.com/go-task/task/issues/219)).
- We now use some colors on Task output to better distinguish message types - commands are green, errors are red, etc ([#207](https://github.com/go-task/task/issues/207)).

## v2.8.1 - 2020-05-20

- Fix error code for the `--help` flag ([#300](https://github.com/go-task/task/issues/300), [#330](https://github.com/go-task/task/issues/330)).
- Print version to stdout instead of stderr ([#299](https://github.com/go-task/task/issues/299), [#329](https://github.com/go-task/task/issues/329)).
- Supress `context` errors when using the `--watch` flag ([#313](https://github.com/go-task/task/issues/313), [#317](https://github.com/go-task/task/issues/317)).
- Support templating on description ([#276](https://github.com/go-task/task/issues/276), [#283](https://github.com/go-task/task/issues/283)).

## v2.8.0 - 2019-12-07

- Add `--parallel` flag (alias `-p`) to run tasks given by the command line in parallel ([#266](https://github.com/go-task/task/issues/266)).
- Fixed bug where calling the `task` CLI only informing global vars would not execute the `default` task.
- Add hability to silent all tasks by adding `silent: true` a the root of the Taskfile.

## v2.7.1 - 2019-11-10

- Fix error being raised when `exit 0` was called ([#251](https://github.com/go-task/task/issues/251)).

## v2.7.0 - 2019-09-22

- Fixed panic bug when assigning a global variable ([#229](https://github.com/go-task/task/issues/229), [#243](https://github.com/go-task/task/issues/243)).
- A task with `method: checksum` will now re-run if generated files are deleted ([#228](https://github.com/go-task/task/issues/228), [#238](https://github.com/go-task/task/issues/238)).

## v2.6.0 - 2019-07-21

- Fixed some bugs regarding minor version checks on `version:`.
- Add `preconditions:` to task ([#205](https://github.com/go-task/task/issues/205)).
- Create directory informed on `dir:` if it doesn't exist ([#209](https://github.com/go-task/task/issues/209), [#211](https://github.com/go-task/task/issues/211)).
- We now have a `--taskfile` flag (alias `-t`), which can be used to run another Taskfile (other than the default `Taskfile.yml`) ([#221](https://github.com/go-task/task/issues/221)).
- It's now possible to install Task using Homebrew on Linux ([go-task/homebrew-tap\[#1\](https://github.com/go-task/task/issues/1)](https://github.com/go-task/homebrew-tap/pull/1)).

## v2.5.2 - 2019-05-11

- Reverted YAML upgrade due issues with CRLF on Windows ([#201](https://github.com/go-task/task/issues/201), [go-yaml/yaml\[#450\](https://github.com/go-task/task/issues/450)](https://github.com/go-yaml/yaml/issues/450)).
- Allow setting global variables through the CLI ([#192](https://github.com/go-task/task/issues/192)).

## 2.5.1 - 2019-04-27

- Fixed some issues with interactive command line tools, where sometimes the output were not being shown, and similar issues ([#114](https://github.com/go-task/task/issues/114), [#190](https://github.com/go-task/task/issues/190), [#200](https://github.com/go-task/task/issues/200)).
- Upgraded [go-yaml/yaml](https://github.com/go-yaml/yaml) from v2 to v3.

## v2.5.0 - 2019-03-16

- We moved from the taskfile.org domain to the new fancy taskfile.dev domain. While stuff is being redirected, we strongly recommend to everyone that use [this install script](https://taskfile.dev/#/installation?id=install-script) to use the new taskfile.dev domain on scripts from now on.
- Fixed to the ZSH completion ([#182](https://github.com/go-task/task/issues/182)).
- Add [`--summary` flag along with `summary:` task attribute](https://taskfile.org/#/usage?id=display-summary-of-task) ([#180](https://github.com/go-task/task/issues/180)).

## v2.4.0 - 2019-02-21

- Allow calling a task of the root Taskfile from an included Taskfile by prefixing it with `:` ([#161](https://github.com/go-task/task/issues/161), [#172](https://github.com/go-task/task/issues/172)).
- Add flag to override the `output` option ([#173](https://github.com/go-task/task/issues/173)).
- Fix bug where Task was persisting the new checksum on the disk when the Dry Mode is enabled ([#166](https://github.com/go-task/task/issues/166)).
- Fix file timestamp issue when the file name has spaces ([#176](https://github.com/go-task/task/issues/176)).
- Mitigating path expanding issues on Windows ([#170](https://github.com/go-task/task/issues/170)).

## v2.3.0 - 2019-01-02

- On Windows, Task can now be installed using [Scoop](https://scoop.sh/) ([#152](https://github.com/go-task/task/issues/152)).
- Fixed issue with file/directory globing ([#153](https://github.com/go-task/task/issues/153)).
- Added ability to globally set environment variables ([#138](https://github.com/go-task/task/issues/138), [#159](https://github.com/go-task/task/issues/159)).

## v2.2.1 - 2018-12-09

- This repository now uses Go Modules ([#143](https://github.com/go-task/task/issues/143)). We'll still keep the `vendor` directory in sync for some time, though;
- Fixing a bug when the Taskfile has no tasks but includes another Taskfile ([#150](https://github.com/go-task/task/issues/150));
- Fix a bug when calling another task or a dependency in an included Taskfile ([#151](https://github.com/go-task/task/issues/151)).

## v2.2.0 - 2018-10-25

- Added support for [including other Taskfiles](https://taskfile.org/#/usage?id=including-other-taskfiles) ([#98](https://github.com/go-task/task/issues/98))
  - This should be considered experimental. For now, only including local files is supported, but support for including remote Taskfiles is being discussed. If you have any feedback, please comment on [#98](https://github.com/go-task/task/issues/98).
- Task now have a dedicated documentation site: https://taskfile.org
  - Thanks to [Docsify](https://docsify.js.org/) for making this pretty easy. To check the source code, just take a look at the [docs](https://github.com/go-task/task/tree/main/docs) directory of this repository. Contributions to the documentation is really appreciated.

## v2.1.1 - 2018-09-17

- Fix suggestion to use `task --init` not being shown anymore (when a `Taskfile.yml` is not found)
- Fix error when using checksum method and no file exists for a source glob ([#131](https://github.com/go-task/task/issues/131))
- Fix signal handling when the `--watch` flag is given ([#132](https://github.com/go-task/task/issues/132))

## v2.1.0 - 2018-08-19

- Add a `ignore_error` option to task and command ([#123](https://github.com/go-task/task/issues/123))
- Add a dry run mode (`--dry` flag) ([#126](https://github.com/go-task/task/issues/126))

## v2.0.3 - 2018-06-24

- Expand environment variables on "dir", "sources" and "generates" ([#116](https://github.com/go-task/task/issues/116))
- Fix YAML merging syntax ([#112](https://github.com/go-task/task/issues/112))
- Add ZSH completion ([#111](https://github.com/go-task/task/issues/111))
- Implement new `output` option. Please check out the [documentation](https://github.com/go-task/task#output-syntax)

## v2.0.2 - 2018-05-01

- Fix merging of YAML anchors ([#112](https://github.com/go-task/task/issues/112))

## v2.0.1 - 2018-03-11

- Fixes panic on `task --list`

## v2.0.0 - 2018-03-08

Version 2.0.0 is here, with a new Taskfile format.

Please, make sure to read the [Taskfile versions](https://github.com/go-task/task/blob/main/TASKFILE_VERSIONS.md) document, since it describes in depth what changed for this version.

- New Taskfile version 2 ([#77](https://github.com/go-task/task/issues/77))
- Possibility to have global variables in the `Taskfile.yml` instead of `Taskvars.yml` ([#66](https://github.com/go-task/task/issues/66))
- Small improvements and fixes

## v1.4.4 - 2017-11-19

- Handle SIGINT and SIGTERM ([#75](https://github.com/go-task/task/issues/75));
- List: print message with there's no task with description;
- Expand home dir ("~" symbol) on paths ([#74](https://github.com/go-task/task/issues/74));
- Add Snap as an installation method;
- Move examples to its own repo;
- Watch: also walk on tasks called on on "cmds", and not only on "deps";
- Print logs to stderr instead of stdout ([#68](https://github.com/go-task/task/issues/68));
- Remove deprecated `set` keyword;
- Add checksum based status check, alternative to timestamp based.

## v1.4.3 - 2017-09-07

- Allow assigning variables to tasks at run time via CLI ([#33](https://github.com/go-task/task/issues/33))
- Added suport for multiline variables from sh ([#64](https://github.com/go-task/task/issues/64))
- Fixes env: remove square braces and evaluate shell ([#62](https://github.com/go-task/task/issues/62))
- Watch: change watch library and few fixes and improvements
- When use watching, cancel and restart long running process on file change ([#59](https://github.com/go-task/task/issues/59) and [#60](https://github.com/go-task/task/issues/60))

## v1.4.2 - 2017-07-30

- Flag to set directory of execution
- Always echo command if is verbose mode
- Add silent mode to disable echoing of commands
- Fixes and improvements of variables ([#56](https://github.com/go-task/task/issues/56))

## v1.4.1 - 2017-07-15

- Allow use of YAML for dynamic variables instead of $ prefix
  - `VAR: {sh: echo Hello}` instead of `VAR: $echo Hello`
- Add `--list` (or `-l`) flag to print existing tasks
- OS specific Taskvars file (e.g. `Taskvars_windows.yml`, `Taskvars_linux.yml`, etc)
- Consider task up-to-date on equal timestamps ([#49](https://github.com/go-task/task/issues/49))
- Allow absolute path in generates section ([#48](https://github.com/go-task/task/issues/48))
- Bugfix: allow templating when calling deps ([#42](https://github.com/go-task/task/issues/42))
- Fix panic for invalid task in cyclic dep detection
- Better error output for dynamic variables in Taskvars.yml ([#41](https://github.com/go-task/task/issues/41))
- Allow template evaluation in parameters

## v1.4.0 - 2017-07-06

- Cache dynamic variables
- Add verbose mode (`-v` flag)
- Support to task parameters (overriding vars) ([#31](https://github.com/go-task/task/issues/31)) ([#32](https://github.com/go-task/task/issues/32))
- Print command, also when "set:" is specified ([#35](https://github.com/go-task/task/issues/35))
- Improve task command help text ([#35](https://github.com/go-task/task/issues/35))

## v1.3.1 - 2017-06-14

- Fix glob not working on commands ([#28](https://github.com/go-task/task/issues/28))
- Add ExeExt template function
- Add `--init` flag to create a new Taskfile
- Add status option to prevent task from running ([#27](https://github.com/go-task/task/issues/27))
- Allow interpolation on `generates` and `sources` attributes ([#26](https://github.com/go-task/task/issues/26))

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
- Tasks can now have a description, and help will print them ([#10](https://github.com/go-task/task/issues/10))
- Task dependencies now run concurrently
- Support for a default task ([#16](https://github.com/go-task/task/issues/16))

## v1.1.0 - 2017-03-08

- Support for YAML, TOML and JSON ([#1](https://github.com/go-task/task/issues/1))
- Support running command in another directory ([#4](https://github.com/go-task/task/issues/4))
- `--force` or `-f` flag to force execution of task even when it's up-to-date
- Detection of cyclic dependencies ([#5](https://github.com/go-task/task/issues/5))
- Support for variables ([#6](https://github.com/go-task/task/issues/6), [#9](https://github.com/go-task/task/issues/9), [#14](https://github.com/go-task/task/issues/14))
- Operation System specific commands and variables ([#13](https://github.com/go-task/task/issues/13))

## v1.0.0 - 2017-02-28

- Add LICENSE file
