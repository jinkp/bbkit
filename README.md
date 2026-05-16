[![CI](https://img.shields.io/badge/CI-GitHub_Actions-blue)](./.github/workflows/ci.yml) [![npm version](https://img.shields.io/npm/v/bbkit-cli)](https://www.npmjs.com/package/bbkit-cli) [![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](./LICENSE)

# bbkit-cli

Modern command-line workflows for Bitbucket Cloud.

## Installation

```bash
npm install -g bbkit-cli
```

## Quick start

```bash
bbk auth login
bbk --version
bbk repo list --workspace myworkspace
bbk pr create --workspace myworkspace --repo my-repo --source feature/my-change --target main --title "Add my change"
bbk pr view 123
bbk pr status 123
bbk pr commits 123
bbk pr reviewers 123
bbk pr tasks 123
bbk pr task create 123 --message "Address review feedback"
bbk pr task resolve 123 456
bbk pr checks 123
bbk pr checkout 123
bbk pr open 123
bbk pr approve 123
bbk pr comment 123 --message "Revisa este caso borde"
bbk pr comments 123
bbk pr files 123
bbk pr diff 123
bbk pr merge 123 --yes --strategy squash --message "Merge feature X"
bbk pr decline 123 --reason "Faltan pruebas"
```

## Command reference

| Command | Description | Flags |
|---|---|---|
| `bbk --version` / `bbk -V` | Print the installed CLI version. | None |
| `bbk version` | Print the installed CLI version, with optional runtime details. | `--json` |
| `bbk auth login` | Authenticate with Bitbucket Cloud using an API token. | None |
| `bbk auth status` | Show the current authentication state. | None |
| `bbk auth logout` | Remove stored credentials. | None |
| `bbk repo list` | List repositories for a workspace. | `--workspace <name>`, `--json` |
| `bbk pr list` | List pull requests. Defaults to open PRs and supports server-side filters. | `--repo <slug>`, `--workspace <name>`, `--json`, `--state <state>`, `--all`, `--author <uuid>`, `--reviewer <uuid>`, `--source <branch>`, `--target <branch>` |
| `bbk pr create` | Create a pull request. | `--source <branch>` (required), `--target <branch>` (required), `--title <title>` (required), `--draft`, `--repo <slug>`, `--workspace <name>` |
| `bbk pr view <id>` | Show pull request metadata, branches, author, reviewers, participants, and URL. | `--repo <slug>`, `--workspace <name>`, `--json` |
| `bbk pr status <id>` | Show a compact PR metadata summary: state, draft/queued flags, counts, and approvals. | `--repo <slug>`, `--workspace <name>`, `--json` |
| `bbk pr commits <id>` | List commits associated with a pull request. | `--repo <slug>`, `--workspace <name>`, `--json` |
| `bbk pr reviewers <id>` | List reviewers and participants with approval/state information when available. | `--repo <slug>`, `--workspace <name>`, `--json` |
| `bbk pr tasks <id>` | List pull request tasks. | `--repo <slug>`, `--workspace <name>`, `--json` |
| `bbk pr task create <id>` | Create a pull request task. | `--message <text>` (required), `--repo <slug>`, `--workspace <name>` |
| `bbk pr task resolve <id> <taskId>` | Resolve a pull request task. | `--repo <slug>`, `--workspace <name>` |
| `bbk pr checks <id>` | List commit/build statuses for a pull request. | `--repo <slug>`, `--workspace <name>`, `--json` |
| `bbk pr checkout <id>` | Fetch and check out a pull request source branch locally. | `--repo <slug>`, `--workspace <name>` |
| `bbk pr open <id>` | Open a pull request in the default browser. | `--repo <slug>`, `--workspace <name>` |
| `bbk pr approve <id>` | Approve a pull request. | `--repo <slug>`, `--workspace <name>` |
| `bbk pr comment <id>` | Post a comment on a pull request. | `--message <text>` (required), `--repo <slug>`, `--workspace <name>` |
| `bbk pr comments <id>` | List pull request comments. | `--repo <slug>`, `--workspace <name>`, `--json` |
| `bbk pr files <id>` | List files changed in a pull request. | `--repo <slug>`, `--workspace <name>`, `--json` |
| `bbk pr diff <id>` | Show the raw unified diff for a pull request. | `--repo <slug>`, `--workspace <name>` |
| `bbk pr merge <id>` | Merge a pull request after confirmation. | `--yes`, `--strategy <strategy>`, `--message <text>`, `--repo <slug>`, `--workspace <name>` |
| `bbk pr decline <id>` | Post the reason as a pull request comment, then decline the pull request. | `--reason <text>` (required), `--repo <slug>`, `--workspace <name>` |
| `bbk branch list` | List repository branches. | `--repo <slug>`, `--workspace <name>`, `--json` |
| `bbk branch stale` | List branches older than a given age. | `--days <n>` (required), `--repo <slug>`, `--workspace <name>`, `--json` |
| `bbk pipeline list` | List recent pipelines. | `--branch <name>`, `--repo <slug>`, `--workspace <name>`, `--json` |
| `bbk pipeline run` | Run a pipeline for a branch. | `--branch <name>` (required), `--repo <slug>`, `--workspace <name>` |

Use `bbk --help` or `bbk <command> --help` for the generated Commander help output.

## Authentication

`bbk auth login` prompts for a Bitbucket username and API token, validates them against the Bitbucket `/user` endpoint, and stores them in the OS keychain when supported.

For CI or other non-interactive environments, set environment variables instead:

```bash
export BITBUCKET_USERNAME="your-username"
export BITBUCKET_API_TOKEN="your-api-token"
```

Environment credentials take precedence over stored credentials.

## Configuration

Workspace resolution follows this order:

1. `--workspace <name>`
2. `BITBUCKET_WORKSPACE`
3. Stored local config (when available)
4. Bitbucket git remote inference from `origin`

If you work with one workspace most of the time, set `BITBUCKET_WORKSPACE` in your shell profile or CI environment so commands like `bbk repo list` do not need the flag every time.

Repository-scoped commands also try to infer `--repo` from the current git remote when you run them inside a Bitbucket repository.

## Pull request list filters

`bbk pr list` queries open pull requests by default. To include other states, pass repeatable `--state` flags with one of `OPEN`, `MERGED`, `DECLINED`, or `SUPERSEDED`:

```bash
bbk pr list --state merged --state declined
```

Use `--all` to request all supported states explicitly. `--all` cannot be combined with `--state`.

Additional server-side filters are available for participants and branches:

```bash
bbk pr list --author '{author-uuid}' --reviewer '{reviewer-uuid}' --source feature/my-change --target main
```

Prefer Bitbucket UUID values for `--author` and `--reviewer`; display names and nicknames are not reliable query identifiers in Bitbucket Cloud.

## Pull request merge safety

`bbk pr merge <id>` asks for confirmation before calling Bitbucket. Use `--yes` only for automation. Bitbucket Cloud supports the `message` field for the resulting commit message and these merge strategies: `merge_commit`, `squash`, `fast_forward`, `squash_fast_forward`, `rebase_fast_forward`, and `rebase_merge`.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for development setup, commit rules, and pull request expectations.

## License

MIT — see [LICENSE](./LICENSE).
