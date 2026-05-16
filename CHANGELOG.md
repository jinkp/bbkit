# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Added standard `bbk --version` / `bbk -V` output and an explicit `bbk version --json` command for runtime version details.
- Added pull request descriptions when creating PRs with `bbk pr create --description`.
- Added `--description-file` support to read PR descriptions from markdown files.
- Added `bbk pr update` to update an existing pull request title, description, or target branch.
- Added `bbk pr checkout <id>` to fetch and check out a pull request source branch locally.
- Added `bbk pr open <id>` to open a pull request in the default browser.
- Added `bbk pr approve <id>` to approve pull requests.
- Added `bbk pr comment <id> --message <text>` to post pull request comments.
- Added `bbk pr comments <id>` to list pull request comments with table and JSON output.
- Added `bbk pr files <id>` to list pull request changed files with table and JSON output.
- Added `bbk pr diff <id>` to print raw unified pull request diffs.
- Added `bbk pr decline <id> --reason <text>` to comment with the reason before declining pull requests.
- Added `bbk pr merge <id>` with confirmation, `--yes`, `--strategy`, and `--message` support.

### Changed

- Extended the Bitbucket client and PR service to support updating pull requests through the Bitbucket API.
- Extended the PR service to fetch individual pull requests for local checkout workflows.
- Added a browser-opening service boundary with URL validation for pull request open workflows.
- Extended the PR service with Bitbucket approval, comment, and decline API calls for review management.
- Extended the PR service with Bitbucket Cloud pull request merge API support.
- Extended the PR service with Bitbucket Cloud pull request diffstat pagination support.
- Extended the Bitbucket client and PR service with text-response support for pull request diffs.
