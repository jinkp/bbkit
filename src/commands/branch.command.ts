import { Command } from 'commander'
import { BitbucketApiError, CliError, mapApiErrorToCliError } from '../errors/cli.errors.js'
import { formatJson } from '../formatters/json.formatter.js'
import { formatTable } from '../formatters/table.formatter.js'
import { BranchService } from '../services/branch.service.js'
import { logger } from '../utils/logger.js'
import { createBitbucketClient, resolveWorkspaceRepo } from './command.helpers.js'

const DAY_IN_MS = 24 * 60 * 60 * 1000

function getAgeInDays(value: string): number {
  return Math.max(0, Math.floor((Date.now() - new Date(value).getTime()) / DAY_IN_MS))
}

export const branchCommand = new Command('branch').description('Manage Bitbucket branches')

branchCommand
  .command('list')
  .description('List repository branches')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .option('--json', 'Output raw JSON')
  .action(async (opts: { repo?: string; workspace?: string; json?: boolean }) => {
    try {
      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const service = new BranchService(client)
      const branches = await service.list(workspace, repoSlug)

      if (opts.json) {
        logger.log(formatJson(branches))
        return
      }

      logger.log(
        formatTable(
          [
            { header: 'Name', key: 'name' },
            { header: 'Last Commit', key: 'hash' },
            { header: 'Author', key: 'author' },
            { header: 'Age (days)', key: 'age' },
          ],
          branches.map((branch) => ({
            name: branch.name,
            hash: branch.target.hash.slice(0, 12),
            author: branch.target.author.user?.display_name ?? '-',
            age: getAgeInDays(branch.target.date),
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

branchCommand
  .command('stale')
  .description('List stale branches older than N days')
  .requiredOption('--days <n>', 'Minimum branch age in days', (value) => Number.parseInt(value, 10))
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .option('--json', 'Output raw JSON')
  .action(async (opts: { days: number; repo?: string; workspace?: string; json?: boolean }) => {
    try {
      if (Number.isNaN(opts.days) || opts.days < 0) {
        throw new Error('invalid-days')
      }

      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const service = new BranchService(client)
      const branches = await service.stale(workspace, repoSlug, opts.days)

      if (opts.json) {
        logger.log(formatJson(branches))
        return
      }

      logger.log(
        formatTable(
          [
            { header: 'Name', key: 'name' },
            { header: 'Last Commit', key: 'hash' },
            { header: 'Author', key: 'author' },
            { header: 'Age (days)', key: 'age' },
          ],
          branches.map((branch) => ({
            name: branch.name,
            hash: branch.target.hash.slice(0, 12),
            author: branch.target.author.user?.display_name ?? '-',
            age: getAgeInDays(branch.target.date),
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

      if (err instanceof Error && err.message === 'invalid-days') {
        throw new CliError('The --days option must be a non-negative number.', 1)
      }

      throw new CliError('An unexpected error occurred.', 2)
    }
  })
