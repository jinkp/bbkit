# Contributing

Thanks for contributing to `bbkit-cli`.

## Prerequisites

- Node.js 20+
- npm

## Development setup

```bash
git clone <your-fork-or-repository-url>
cd bbkit
npm install
npm test
```

You should also run `npm run typecheck` and `npm run lint` before opening a pull request.

## Commit conventions

This project requires Conventional Commits. Examples:

- `feat: add pipeline run command`
- `fix: map bitbucket 404 errors to cli message`
- `docs: add release workflow documentation`

## Pull request process

1. Create a branch from `main`.
2. Make your changes with tests when relevant.
3. Open a pull request.
4. Ensure CI passes before requesting or merging review.

## Code style

ESLint and Prettier are enforced for the project. Keep TypeScript strict-mode clean and avoid bypassing lint or formatting rules in submitted changes.
