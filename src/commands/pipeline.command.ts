import { Command } from 'commander'
import { BitbucketApiError, CliError, mapApiErrorToCliError } from '../errors/cli.errors.js'
import { formatJson } from '../formatters/json.formatter.js'
import { formatTable } from '../formatters/table.formatter.js'
import { PipelineService } from '../services/pipeline.service.js'
import { logger } from '../utils/logger.js'
import { createBitbucketClient, resolveWorkspaceRepo } from './command.helpers.js'

function formatDuration(seconds?: number): string {
  if (seconds == null) {
    return '-'
  }

  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60

  return `${minutes}m ${remainingSeconds}s`
}

function formatDate(value: string): string {
  return new Date(value).toISOString().replace('T', ' ').slice(0, 16)
}

function getPipelineStatus(state: { name: string; result?: { name: string } }): string {
  return state.result?.name ?? state.name
}

export const pipelineCommand = new Command('pipeline').description('Manage Bitbucket pipelines')

pipelineCommand
  .command('list')
  .description('List recent pipelines')
  .option('--branch <name>', 'Filter by branch name')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .option('--json', 'Output raw JSON')
  .action(async (opts: { branch?: string; repo?: string; workspace?: string; json?: boolean }) => {
    try {
      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const service = new PipelineService(client)
      const pipelines = await service.list(workspace, repoSlug, opts.branch)

      if (opts.json) {
        logger.log(formatJson(pipelines))
        return
      }

      logger.log(
        formatTable(
          [
            { header: '#', key: 'buildNumber' },
            { header: 'Status', key: 'status' },
            { header: 'Branch', key: 'branch' },
            { header: 'Duration', key: 'duration' },
            { header: 'Created', key: 'created' },
          ],
          pipelines.map((pipeline) => ({
            buildNumber: pipeline.build_number,
            status: getPipelineStatus(pipeline.state),
            branch: pipeline.target.ref_name ?? '-',
            duration: formatDuration(pipeline.duration_in_seconds),
            created: formatDate(pipeline.created_on),
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

pipelineCommand
  .command('run')
  .description('Run a pipeline for a branch')
  .requiredOption('--branch <name>', 'Branch name to build')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .action(async (opts: { branch: string; repo?: string; workspace?: string }) => {
    try {
      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const service = new PipelineService(client)
      const pipeline = await service.run(workspace, repoSlug, opts.branch)

      logger.success(`Pipeline #${pipeline.build_number} started`)
      logger.log(`URL: ${pipeline.links.self.href}`)
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
