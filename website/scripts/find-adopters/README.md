# find-adopters

A small Go tool that scans GitHub for every public repository containing a
`Taskfile.yml` or `Taskfile.yaml` and produces a ranked list of adopter
candidates for the [taskfile.dev](https://taskfile.dev) "Used by" section.

## How it works

GitHub Code Search caps at 1000 results per query and only accepts a narrow
set of qualifiers alongside `filename:` — notably `stars:`, `language:`, and
`pushed:` don't combine, and `size:` does but its `total_count` isn't monotone
as ranges shrink, which makes partitioning unreliable. So the tool takes a
pragmatic two-pronged approach:

1. **Global best-match pagination** — paginate `filename:Taskfile.yml`,
   `Taskfile.yaml`, `Taskfile.dist.yml`, and `Taskfile.dist.yaml` directly up
   to the 1000-result cap. Captures the top ~900 best-ranked hits per variant.
2. **Per-org scan** — iterate a built-in list of ~100 well-known organizations
   (hyperscalers, OSS vendors, DevOps platforms, etc.) with
   `filename:Taskfile.yml org:<name>`. Captures every Taskfile inside those
   orgs even when their repos don't rank in the global top.

The union is deduplicated and enriched via batched GraphQL calls (stars,
description, owner type, language, topics), then sorted by stars.

A full scan typically takes 15-25 minutes — about 120 Code Search calls at the
10 req/min authenticated rate limit, plus a handful of GraphQL batches.

### Coverage caveat

GitHub's hard 1000-result cap on the Code Search API means this tool cannot
enumerate every Taskfile on GitHub — only the best-ranked slice plus the
curated orgs. For truly exhaustive coverage, consider
[GH Archive](https://www.gharchive.org/) or the BigQuery public GitHub
dataset, which are out of scope here.

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
