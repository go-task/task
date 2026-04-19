# find-adopters

A small Go tool that scans GitHub for every public repository containing a
`Taskfile.yml` or `Taskfile.yaml` and produces a ranked list of adopter
candidates for the [taskfile.dev](https://taskfile.dev) "Used by" section.

## How it works

GitHub Code Search caps results at 1000 per query. `find-adopters` partitions
queries by **star bucket** (and, for the 0-star bucket, by pushed-year) so
every sub-query stays under the cap. Each unique repo is then enriched via a
single batched GraphQL call (stars, description, owner type, language, topics)
and sorted by popularity.

The full scan typically takes 15-30 minutes, mostly spent respecting the
Code Search rate limit (30 req/min).

## Usage

```sh
# From this directory:
go run . -v                           # full scan → adopters-scan.tsv
go run . --min-stars 100 -v           # only >=100 stars
go run . --owner-type org --json -o orgs.json
```

Or from the website root (if the Taskfile task is installed):

```sh
task find-adopters -- --min-stars 100 -v
```

## Flags

| Flag                  | Default              | Description                          |
| --------------------- | -------------------- | ------------------------------------ |
| `-o`                  | `adopters-scan.tsv`  | output path                          |
| `--json`              | `false`              | emit JSON instead of TSV             |
| `--min-stars`         | `0`                  | filter results below threshold       |
| `--include-forks`     | `false`              | include forked repos                 |
| `--include-archived`  | `false`              | include archived repos               |
| `--owner-type`        | `any`                | `org`, `user`, or `any`              |
| `-v`                  | `false`              | verbose progress logging             |

## Auth

A GitHub token is required. In order of precedence:

1. `GITHUB_TOKEN` env var
2. `gh auth token` (requires the [GitHub CLI](https://cli.github.com/))

The token needs no special scopes for public data.

## Output (TSV)

```
stars	full_name	owner_type	language	url	description
13619	OJ/gobuster	User	Go	https://github.com/OJ/gobuster	Directory/File...
10918	FerretDB/FerretDB	Organization	Go	https://github.com/FerretDB/FerretDB	A truly Open Source MongoDB alternative
...
```
