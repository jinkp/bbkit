import { Command } from 'commander'
import { configStore } from '../config/config.store.js'
import { CliError, BitbucketApiError, mapApiErrorToCliError } from '../errors/cli.errors.js'
import { formatJson } from '../formatters/json.formatter.js'
import { formatTable } from '../formatters/table.formatter.js'
import { RepoService } from '../services/repo.service.js'
import { inferFromGitRemote } from '../utils/git-remote.js'
import { logger } from '../utils/logger.js'
import { createBitbucketClient } from './command.helpers.js'

function resolveWorkspace(workspace?: string): string {
  const resolved = workspace ?? configStore.getWorkspace() ?? inferFromGitRemote()?.workspace

  if (!resolved) {
    throw new CliError('No workspace configured. Use --workspace or run `bbk auth login`.', 1)
  }

  return resolved
}

function formatDate(value: string): string {
  return value.split('T')[0] ?? value
}

export const repoCommand = new Command('repo').description('Manage Bitbucket repositories')

repoCommand
  .command('list')
  .description('List repositories in a workspace')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .option('--json', 'Output raw JSON')
  .action(async (opts: { workspace?: string; json?: boolean }) => {
    try {
      const workspace = resolveWorkspace(opts.workspace)
      const client = await createBitbucketClient()
      const service = new RepoService(client)
      const repositories = await service.list(workspace)

      if (opts.json) {
        logger.log(formatJson(repositories))
        return
      }

      logger.log(
        formatTable(
          [
            { header: 'Name', key: 'name' },
            { header: 'Description', key: 'description' },
            { header: 'Language', key: 'language' },
            { header: 'Updated', key: 'updated' },
          ],
          repositories.map((repository) => ({
            name: repository.name,
            description: repository.description || '-',
            language: repository.language ?? '-',
            updated: formatDate(repository.updated_on),
          })),
        ),
      )
    } catch (err) {
      if (err instanceof CliError) {
        throw err
      }

      if (err instanceof BitbucketApiError) {
        throw mapApiErrorToCliError(err)
      }

      throw new CliError('An unexpected error occurred.', 2)
    }
  })
