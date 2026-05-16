import { BitbucketClient } from '../client/bitbucket.client.js'
import { configStore } from '../config/config.store.js'
import { CliError } from '../errors/cli.errors.js'
import { authService } from '../services/auth.service.js'
import { inferFromGitRemote } from '../utils/git-remote.js'

export async function createBitbucketClient(): Promise<BitbucketClient> {
  const credentials = await authService.getCredentialsOrFail()

  return new BitbucketClient(credentials.username, credentials.apiToken)
}

export function resolveWorkspaceRepo(opts: {
  workspace?: string
  repo?: string
}): { workspace: string; repoSlug: string } {
  const fromRemote = inferFromGitRemote()
  const workspace = opts.workspace ?? configStore.getWorkspace() ?? fromRemote?.workspace
  const repoSlug = opts.repo ?? fromRemote?.repoSlug

  if (!workspace) {
    throw new CliError(
      [
        'No workspace configured.',
        'Options:',
        '  1. Run `bbk auth login` and enter your workspace slug',
        '  2. Use the --workspace <slug> flag',
        '  3. Set the BITBUCKET_WORKSPACE environment variable',
        '  4. Run from inside a git repo with a Bitbucket remote',
        '',
        'Your workspace slug is the identifier in your Bitbucket URL:',
        '  https://bitbucket.org/{workspace}/...',
      ].join('\n'),
      1,
    )
  }

  if (!repoSlug) {
    throw new CliError('No repository found. Use --repo or run from a git repository.', 1)
  }

  return { workspace, repoSlug }
}
