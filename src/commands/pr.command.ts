import { readFileSync } from 'node:fs'
import { Command } from 'commander'
import * as p from '@clack/prompts'
import { BitbucketApiError, CliError, mapApiErrorToCliError } from '../errors/cli.errors.js'
import { formatJson } from '../formatters/json.formatter.js'
import { formatTable } from '../formatters/table.formatter.js'
import { BrowserOpenService } from '../services/browser-open.service.js'
import { GitCheckoutService } from '../services/git-checkout.service.js'
import {
  PrService,
  type PrListOptions,
  type PrMergePayload,
  type PrMergeStrategy,
  type PrUpdatePayload,
} from '../services/pr.service.js'
import type {
  BitbucketAccount,
  BitbucketCommit,
  BitbucketCommitStatus,
  BitbucketPullRequest,
  BitbucketPullRequestComment,
  BitbucketPullRequestDiffstat,
  BitbucketPullRequestState,
  BitbucketPullRequestTask,
} from '../types/bitbucket.types.js'
import { logger } from '../utils/logger.js'
import { createBitbucketClient, resolveWorkspaceRepo } from './command.helpers.js'

function resolveDescription(description?: string, descriptionFile?: string): string | undefined {
  if (descriptionFile) {
    try {
      return readFileSync(descriptionFile, 'utf-8')
    } catch {
      throw new CliError(`Could not read description file: ${descriptionFile}`, 1)
    }
  }
  return description
}

function formatDate(value: string): string {
  return new Date(value).toISOString().slice(0, 10)
}

function parsePullRequestId(id: string): number {
  const prId = Number(id)
  if (!Number.isInteger(prId) || prId <= 0) {
    throw new CliError(`Invalid pull request ID: ${id}`, 1)
  }
  return prId
}

function parseTaskId(id: string): number {
  const taskId = Number(id)
  if (!Number.isInteger(taskId) || taskId <= 0) {
    throw new CliError(`Invalid task ID: ${id}`, 1)
  }
  return taskId
}

function resolveTaskMessage(message: string): string {
  const trimmedMessage = message.trim()
  if (trimmedMessage.length === 0) {
    throw new CliError('Task message cannot be empty.', 1)
  }

  return trimmedMessage
}

function mapPullRequestApiError(err: BitbucketApiError, id: string): CliError {
  if (err.status === 404) {
    return new CliError(`PR #${id} not found.`, 1)
  }

  return mapApiErrorToCliError(err)
}

const SUPPORTED_MERGE_STRATEGIES: readonly PrMergeStrategy[] = [
  'merge_commit',
  'squash',
  'fast_forward',
  'squash_fast_forward',
  'rebase_fast_forward',
  'rebase_merge',
]

const SUPPORTED_PULL_REQUEST_STATES: readonly BitbucketPullRequestState[] = [
  'OPEN',
  'MERGED',
  'DECLINED',
  'SUPERSEDED',
]

type PrListCommandOptions = {
  repo?: string
  workspace?: string
  json?: boolean
  state?: string[]
  all?: boolean
  author?: string
  reviewer?: string
  source?: string
  target?: string
}

function collectRepeatedOption(value: string, previous: string[] = []): string[] {
  return [...previous, value]
}

function parsePullRequestStates(states: string[] | undefined): BitbucketPullRequestState[] {
  const parsedStates: BitbucketPullRequestState[] = []

  for (const state of states ?? []) {
    const normalizedState = state.toUpperCase()
    if (!SUPPORTED_PULL_REQUEST_STATES.includes(normalizedState as BitbucketPullRequestState)) {
      throw new CliError(
        `Invalid pull request state: ${state}. Supported states: ${SUPPORTED_PULL_REQUEST_STATES.join(', ')}.`,
        1,
      )
    }

    if (!parsedStates.includes(normalizedState as BitbucketPullRequestState)) {
      parsedStates.push(normalizedState as BitbucketPullRequestState)
    }
  }

  return parsedStates
}

function resolveOptionalFilter(name: string, value: string | undefined): string | undefined {
  if (value === undefined) {
    return undefined
  }

  const trimmedValue = value.trim()
  if (trimmedValue.length === 0) {
    throw new CliError(`Filter ${name} cannot be empty.`, 1)
  }

  return trimmedValue
}

function resolvePrListOptions(opts: PrListCommandOptions): PrListOptions {
  if (opts.all && opts.state && opts.state.length > 0) {
    throw new CliError('Use either --all or --state, not both.', 1)
  }

  const parsedStates = parsePullRequestStates(opts.state)

  return {
    states: opts.all
      ? [...SUPPORTED_PULL_REQUEST_STATES]
      : parsedStates.length > 0
        ? parsedStates
        : ['OPEN'],
    author: resolveOptionalFilter('author', opts.author),
    reviewer: resolveOptionalFilter('reviewer', opts.reviewer),
    source: resolveOptionalFilter('source', opts.source),
    target: resolveOptionalFilter('target', opts.target),
  }
}

function resetPrListCommandOptions(command: Command): void {
  for (const option of ['json', 'state', 'all', 'author', 'reviewer', 'source', 'target']) {
    command.setOptionValue(option, undefined)
  }
}

function parseMergeStrategy(strategy?: string): PrMergeStrategy | undefined {
  if (!strategy) {
    return undefined
  }

  if (SUPPORTED_MERGE_STRATEGIES.includes(strategy as PrMergeStrategy)) {
    return strategy as PrMergeStrategy
  }

  throw new CliError(
    `Invalid merge strategy: ${strategy}. Supported strategies: ${SUPPORTED_MERGE_STRATEGIES.join(', ')}.`,
    1,
  )
}

function resolveMergeMessage(message?: string): string | undefined {
  if (message === undefined) {
    return undefined
  }

  const trimmedMessage = message.trim()
  if (trimmedMessage.length === 0) {
    throw new CliError('Merge message cannot be empty.', 1)
  }

  return trimmedMessage
}

function resolveCommentMessage(message: string): string {
  const trimmedMessage = message.trim()
  if (trimmedMessage.length === 0) {
    throw new CliError('Comment message cannot be empty.', 1)
  }

  return trimmedMessage
}

function formatCommentContent(comment: BitbucketPullRequestComment): { raw: string; rendered: string } {
  return {
    raw: comment.content?.raw ?? '',
    rendered: comment.content?.html ?? '',
  }
}

function formatDiffstatPath(file: BitbucketPullRequestDiffstat): string {
  return file.new?.path ?? file.old?.path ?? 'Unknown'
}

function formatOptionalDiffstatPath(file: BitbucketPullRequestDiffstat): string {
  return file.old?.path ?? ''
}

function formatLineCount(value?: number): string {
  return value === undefined ? '' : String(value)
}

function formatOptionalDate(value?: string): string {
  return value ? formatDate(value) : ''
}

function formatAccount(account?: BitbucketAccount): string {
  return account?.display_name ?? account?.nickname ?? account?.uuid ?? 'Unknown'
}

function formatCommitHash(hash: string): string {
  return hash.slice(0, 7)
}

function formatFirstLine(value?: string): string {
  return value?.split('\n')[0] ?? ''
}

function countApprovedReviewers(pullRequest: BitbucketPullRequest): number {
  return pullRequest.participants?.filter((participant) => participant.approved).length ?? 0
}

function formatReviewerRows(pullRequest: BitbucketPullRequest): Record<string, unknown>[] {
  const reviewers = pullRequest.reviewers ?? []
  const participants = pullRequest.participants ?? []

  return [
    ...reviewers.map((reviewer) => ({
      name: formatAccount(reviewer),
      role: 'reviewer',
      approved: '',
      state: '',
    })),
    ...participants.map((participant) => ({
      name: formatAccount(participant.user),
      role: participant.role ?? 'participant',
      approved: participant.approved === undefined ? '' : participant.approved ? 'yes' : 'no',
      state: participant.state ?? '',
    })),
  ]
}

function formatPullRequestDetails(pullRequest: BitbucketPullRequest): Record<string, unknown>[] {
  return [
    { field: 'ID', value: pullRequest.id },
    { field: 'Title', value: pullRequest.title },
    { field: 'State', value: pullRequest.state },
    { field: 'Author', value: formatAccount(pullRequest.author) },
    {
      field: 'Branches',
      value: `${pullRequest.source.branch.name} → ${pullRequest.destination.branch.name}`,
    },
    { field: 'Created', value: formatOptionalDate(pullRequest.created_on) },
    { field: 'Updated', value: formatOptionalDate(pullRequest.updated_on) },
    { field: 'Reviewers', value: pullRequest.reviewers?.map(formatAccount).join(', ') ?? '' },
    { field: 'Participants', value: pullRequest.participants?.map((p) => formatAccount(p.user)).join(', ') ?? '' },
    { field: 'URL', value: pullRequest.links.html.href },
  ]
}

function formatPullRequestStatus(pullRequest: BitbucketPullRequest): Record<string, unknown>[] {
  return [
    { field: 'State', value: pullRequest.state },
    { field: 'Draft', value: pullRequest.draft === undefined ? '' : pullRequest.draft ? 'yes' : 'no' },
    { field: 'Queued', value: pullRequest.queued === undefined ? '' : pullRequest.queued ? 'yes' : 'no' },
    {
      field: 'Branches',
      value: `${pullRequest.source.branch.name} → ${pullRequest.destination.branch.name}`,
    },
    { field: 'Tasks', value: pullRequest.task_count ?? '' },
    { field: 'Comments', value: pullRequest.comment_count ?? '' },
    { field: 'Approvals', value: `${countApprovedReviewers(pullRequest)}/${pullRequest.participants?.length ?? 0}` },
    { field: 'Updated', value: formatOptionalDate(pullRequest.updated_on) },
  ]
}

function formatCommitRows(commits: BitbucketCommit[]): Record<string, unknown>[] {
  return commits.map((commit) => ({
    hash: formatCommitHash(commit.hash),
    message: formatFirstLine(commit.message),
    author: formatAccount(commit.author?.user) || commit.author?.raw || 'Unknown',
    date: formatOptionalDate(commit.date),
    url: commit.links?.html?.href ?? '',
  }))
}

function formatTaskRows(tasks: BitbucketPullRequestTask[]): Record<string, unknown>[] {
  return tasks.map((task) => ({
    id: task.id,
    state: task.state ?? '',
    content: task.content?.raw ?? '',
    creator: formatAccount(task.creator),
    assignee: task.assignee ? formatAccount(task.assignee) : '',
    created: formatOptionalDate(task.created_on),
    updated: formatOptionalDate(task.updated_on),
  }))
}

function formatCheckRows(checks: BitbucketCommitStatus[]): Record<string, unknown>[] {
  return checks.map((check) => ({
    key: check.key,
    name: check.name ?? '',
    state: check.state ?? '',
    description: check.description ?? '',
    updated: formatOptionalDate(check.updated_on),
    url: check.url ?? '',
  }))
}

export const prCommand = new Command('pr').description('Manage Bitbucket pull requests')

prCommand
  .command('list')
  .description('List open pull requests')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .option('--json', 'Output raw JSON')
  .option('--state <state>', 'Filter by pull request state (repeatable)', collectRepeatedOption)
  .option('--all', 'Include open, merged, declined, and superseded pull requests')
  .option('--author <uuid>', 'Filter by author UUID')
  .option('--reviewer <uuid>', 'Filter by reviewer UUID')
  .option('--source <branch>', 'Filter by source branch name')
  .option('--target <branch>', 'Filter by target branch name')
  .action(async function (this: Command, opts: PrListCommandOptions) {
    try {
      const listOptions = resolvePrListOptions(opts)
      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const service = new PrService(client)
      const pullRequests = await service.list(workspace, repoSlug, listOptions)

      if (opts.json) {
        logger.log(formatJson(pullRequests))
        return
      }

      if (pullRequests.length === 0) {
        logger.log('No pull requests found for the selected filters.')
        return
      }

      logger.log(
        formatTable(
          [
            { header: 'ID', key: 'id' },
            { header: 'Title', key: 'title' },
            { header: 'Author', key: 'author' },
            { header: 'Source → Target', key: 'branches' },
            { header: 'State', key: 'state' },
            { header: 'Updated', key: 'updated' },
          ],
          pullRequests.map((pullRequest) => ({
            id: pullRequest.id,
            title: pullRequest.title,
            author: pullRequest.author.display_name,
            branches: `${pullRequest.source.branch.name} → ${pullRequest.destination.branch.name}`,
            state: pullRequest.state,
            updated: formatDate(pullRequest.updated_on),
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
    } finally {
      resetPrListCommandOptions(this)
    }
  })

prCommand
  .command('view')
  .description('Show pull request details')
  .argument('<id>', 'Pull request ID')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .option('--json', 'Output raw JSON', false)
  .action(async (id: string, opts: { repo?: string; workspace?: string; json?: boolean }) => {
    try {
      const prId = parsePullRequestId(id)
      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const prService = new PrService(client)
      const pullRequest = await prService.get(workspace, repoSlug, prId)

      if (opts.json) {
        logger.log(formatJson(pullRequest))
        return
      }

      logger.log(formatTable([{ header: 'Field', key: 'field' }, { header: 'Value', key: 'value' }], formatPullRequestDetails(pullRequest)))
    } catch (err) {
      if (err instanceof CliError) throw err
      if (err instanceof BitbucketApiError) throw mapPullRequestApiError(err, id)
      throw new CliError('An unexpected error occurred.', 2)
    }
  })

prCommand
  .command('status')
  .description('Show pull request status summary')
  .argument('<id>', 'Pull request ID')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .option('--json', 'Output raw JSON', false)
  .action(async (id: string, opts: { repo?: string; workspace?: string; json?: boolean }) => {
    try {
      const prId = parsePullRequestId(id)
      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const prService = new PrService(client)
      const pullRequest = await prService.get(workspace, repoSlug, prId)

      if (opts.json) {
        logger.log(formatJson(pullRequest))
        return
      }

      logger.log(formatTable([{ header: 'Field', key: 'field' }, { header: 'Value', key: 'value' }], formatPullRequestStatus(pullRequest)))
    } catch (err) {
      if (err instanceof CliError) throw err
      if (err instanceof BitbucketApiError) throw mapPullRequestApiError(err, id)
      throw new CliError('An unexpected error occurred.', 2)
    }
  })

prCommand
  .command('commits')
  .description('List commits on a pull request')
  .argument('<id>', 'Pull request ID')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .option('--json', 'Output raw JSON', false)
  .action(async (id: string, opts: { repo?: string; workspace?: string; json?: boolean }) => {
    try {
      const prId = parsePullRequestId(id)
      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const prService = new PrService(client)
      const commits = await prService.commits(workspace, repoSlug, prId)

      if (opts.json) {
        logger.log(formatJson(commits))
        return
      }

      if (commits.length === 0) {
        logger.log(`PR #${prId} has no commits.`)
        return
      }

      logger.log(formatTable([
        { header: 'Hash', key: 'hash' },
        { header: 'Message', key: 'message' },
        { header: 'Author', key: 'author' },
        { header: 'Date', key: 'date' },
        { header: 'URL', key: 'url' },
      ], formatCommitRows(commits)))
    } catch (err) {
      if (err instanceof CliError) throw err
      if (err instanceof BitbucketApiError) throw mapPullRequestApiError(err, id)
      throw new CliError('An unexpected error occurred.', 2)
    }
  })

prCommand
  .command('reviewers')
  .description('List reviewers and participants on a pull request')
  .argument('<id>', 'Pull request ID')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .option('--json', 'Output raw JSON', false)
  .action(async (id: string, opts: { repo?: string; workspace?: string; json?: boolean }) => {
    try {
      const prId = parsePullRequestId(id)
      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const prService = new PrService(client)
      const pullRequest = await prService.get(workspace, repoSlug, prId)
      const rows = formatReviewerRows(pullRequest)

      if (opts.json) {
        logger.log(formatJson({ reviewers: pullRequest.reviewers ?? [], participants: pullRequest.participants ?? [] }))
        return
      }

      if (rows.length === 0) {
        logger.log(`PR #${prId} has no reviewers.`)
        return
      }

      logger.log(formatTable([
        { header: 'Name', key: 'name' },
        { header: 'Role', key: 'role' },
        { header: 'Approved', key: 'approved' },
        { header: 'State', key: 'state' },
      ], rows))
    } catch (err) {
      if (err instanceof CliError) throw err
      if (err instanceof BitbucketApiError) throw mapPullRequestApiError(err, id)
      throw new CliError('An unexpected error occurred.', 2)
    }
  })

prCommand
  .command('tasks')
  .description('List tasks on a pull request')
  .argument('<id>', 'Pull request ID')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .option('--json', 'Output raw JSON', false)
  .action(async (id: string, opts: { repo?: string; workspace?: string; json?: boolean }) => {
    try {
      const prId = parsePullRequestId(id)
      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const prService = new PrService(client)
      const tasks = await prService.tasks(workspace, repoSlug, prId)

      if (opts.json) {
        logger.log(formatJson(tasks))
        return
      }

      if (tasks.length === 0) {
        logger.log(`PR #${prId} has no tasks.`)
        return
      }

      logger.log(formatTable([
        { header: 'ID', key: 'id' },
        { header: 'State', key: 'state' },
        { header: 'Content', key: 'content' },
        { header: 'Creator', key: 'creator' },
        { header: 'Assignee', key: 'assignee' },
        { header: 'Created', key: 'created' },
        { header: 'Updated', key: 'updated' },
      ], formatTaskRows(tasks)))
    } catch (err) {
      if (err instanceof CliError) throw err
      if (err instanceof BitbucketApiError) throw mapPullRequestApiError(err, id)
      throw new CliError('An unexpected error occurred.', 2)
    }
  })

const prTaskCommand = prCommand.command('task').description('Manage pull request tasks')

prTaskCommand
  .command('create')
  .description('Create a task on a pull request')
  .argument('<id>', 'Pull request ID')
  .requiredOption('--message <text>', 'Task message')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .action(async (id: string, opts: { message: string; repo?: string; workspace?: string }) => {
    try {
      const prId = parsePullRequestId(id)
      const message = resolveTaskMessage(opts.message)
      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const prService = new PrService(client)
      const task = await prService.createTask(workspace, repoSlug, prId, message)

      logger.success(`Task #${task.id} created on PR #${prId}: ${task.content?.raw ?? message}`)
    } catch (err) {
      if (err instanceof CliError) throw err
      if (err instanceof BitbucketApiError) throw mapPullRequestApiError(err, id)
      throw new CliError('An unexpected error occurred.', 2)
    }
  })

prTaskCommand
  .command('resolve')
  .description('Resolve a task on a pull request')
  .argument('<id>', 'Pull request ID')
  .argument('<taskId>', 'Task ID')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .action(async (id: string, taskIdArg: string, opts: { repo?: string; workspace?: string }) => {
    try {
      const prId = parsePullRequestId(id)
      const taskId = parseTaskId(taskIdArg)
      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const prService = new PrService(client)
      const task = await prService.resolveTask(workspace, repoSlug, prId, taskId)

      logger.success(`Task #${task.id} resolved on PR #${prId}.`)
    } catch (err) {
      if (err instanceof CliError) throw err
      if (err instanceof BitbucketApiError) throw mapPullRequestApiError(err, id)
      throw new CliError('An unexpected error occurred.', 2)
    }
  })

prCommand
  .command('checks')
  .description('List commit/build statuses for a pull request')
  .argument('<id>', 'Pull request ID')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .option('--json', 'Output raw JSON', false)
  .action(async (id: string, opts: { repo?: string; workspace?: string; json?: boolean }) => {
    try {
      const prId = parsePullRequestId(id)
      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const prService = new PrService(client)
      const checks = await prService.checks(workspace, repoSlug, prId)

      if (opts.json) {
        logger.log(formatJson(checks))
        return
      }

      if (checks.length === 0) {
        logger.log(`PR #${prId} has no checks.`)
        return
      }

      logger.log(formatTable([
        { header: 'Key', key: 'key' },
        { header: 'Name', key: 'name' },
        { header: 'State', key: 'state' },
        { header: 'Description', key: 'description' },
        { header: 'Updated', key: 'updated' },
        { header: 'URL', key: 'url' },
      ], formatCheckRows(checks)))
    } catch (err) {
      if (err instanceof CliError) throw err
      if (err instanceof BitbucketApiError) throw mapPullRequestApiError(err, id)
      throw new CliError('An unexpected error occurred.', 2)
    }
  })

prCommand
  .command('create')
  .description('Create a pull request')
  .requiredOption('--source <branch>', 'Source branch name')
  .requiredOption('--target <branch>', 'Target branch name')
  .requiredOption('--title <title>', 'Pull request title')
  .option('--description <text>', 'Pull request description (markdown supported)')
  .option('--description-file <path>', 'Read description from a markdown file')
  .option('--draft', 'Create PR as draft')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .action(
    async (opts: {
      source: string
      target: string
      title: string
      description?: string
      descriptionFile?: string
      draft?: boolean
      repo?: string
      workspace?: string
    }) => {
      try {
        const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
        const client = await createBitbucketClient()
        const service = new PrService(client)
        const pullRequest = await service.create(workspace, repoSlug, {
          title: opts.title,
          description: resolveDescription(opts.description, opts.descriptionFile),
          source: { branch: { name: opts.source } },
          destination: { branch: { name: opts.target } },
          draft: opts.draft,
        })

        logger.success(`PR created: ${pullRequest.links.html.href}`)
      } catch (err) {
        if (err instanceof CliError) {
          throw err
        }

        if (err instanceof BitbucketApiError) {
          if (err.status === 404) {
            throw new CliError(`Source branch '${opts.source}' not found.`, 1)
          }

          throw mapApiErrorToCliError(err)
        }

        throw new CliError('An unexpected error occurred.', 2)
      }
    },
  )

prCommand
  .command('checkout')
  .description('Check out a pull request source branch locally')
  .argument('<id>', 'Pull request ID')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .action(async (id: string, opts: { repo?: string; workspace?: string }) => {
    try {
      const prId = parsePullRequestId(id)

      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const prService = new PrService(client)
      const gitCheckoutService = new GitCheckoutService()
      const pullRequest = await prService.get(workspace, repoSlug, prId)
      const result = gitCheckoutService.checkoutSourceBranch(pullRequest.source.branch.name)

      logger.success(`Checked out PR #${pullRequest.id} source branch: ${result.branchName}`)
    } catch (err) {
      if (err instanceof CliError) {
        throw err
      }

      if (err instanceof BitbucketApiError) {
        throw mapPullRequestApiError(err, id)
      }

      throw new CliError('An unexpected error occurred.', 2)
    }
  })

prCommand
  .command('open')
  .description('Open a pull request in the default browser')
  .argument('<id>', 'Pull request ID')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .action(async (id: string, opts: { repo?: string; workspace?: string }) => {
    try {
      const prId = parsePullRequestId(id)

      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const prService = new PrService(client)
      const browserOpenService = new BrowserOpenService()
      const pullRequest = await prService.get(workspace, repoSlug, prId)
      const result = browserOpenService.openUrl(pullRequest.links.html.href)

      logger.success(`Opened PR #${pullRequest.id}: ${result.url}`)
    } catch (err) {
      if (err instanceof CliError) {
        throw err
      }

      if (err instanceof BitbucketApiError) {
        throw mapPullRequestApiError(err, id)
      }

      throw new CliError('An unexpected error occurred.', 2)
    }
  })

prCommand
  .command('approve')
  .description('Approve a pull request')
  .argument('<id>', 'Pull request ID')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .action(async (id: string, opts: { repo?: string; workspace?: string }) => {
    try {
      const prId = parsePullRequestId(id)
      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const prService = new PrService(client)

      await prService.approve(workspace, repoSlug, prId)

      logger.success(`Approved PR #${prId}.`)
    } catch (err) {
      if (err instanceof CliError) {
        throw err
      }

      if (err instanceof BitbucketApiError) {
        throw mapPullRequestApiError(err, id)
      }

      throw new CliError('An unexpected error occurred.', 2)
    }
  })

prCommand
  .command('decline')
  .description('Decline a pull request and post the reason as a comment')
  .argument('<id>', 'Pull request ID')
  .requiredOption('--reason <text>', 'Reason to post as a pull request comment before declining')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .action(async (id: string, opts: { reason: string; repo?: string; workspace?: string }) => {
    try {
      const prId = parsePullRequestId(id)
      const reason = opts.reason.trim()
      if (reason.length === 0) {
        throw new CliError('Decline reason cannot be empty.', 1)
      }

      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const prService = new PrService(client)

      await prService.decline(workspace, repoSlug, prId, reason)

      logger.success(`Declined PR #${prId} with reason comment.`)
    } catch (err) {
      if (err instanceof CliError) {
        throw err
      }

      if (err instanceof BitbucketApiError) {
        throw mapPullRequestApiError(err, id)
      }

      throw new CliError('An unexpected error occurred.', 2)
    }
  })

prCommand
  .command('merge')
  .description('Merge a pull request')
  .argument('<id>', 'Pull request ID')
  .option('--yes', 'Skip confirmation prompt for automation', false)
  .option(
    '--strategy <strategy>',
    'Merge strategy: merge_commit, squash, fast_forward, squash_fast_forward, rebase_fast_forward, rebase_merge',
  )
  .option('--message <text>', 'Commit message for the resulting merge commit')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .action(
    async (
      id: string,
      opts: {
        yes?: boolean
        strategy?: string
        message?: string
        repo?: string
        workspace?: string
      },
      command: Command,
    ) => {
      try {
        const confirmedByFlag = command.getOptionValueSource('yes') === 'cli' && opts.yes === true
        const strategy = command.getOptionValueSource('strategy') === 'cli' ? opts.strategy : undefined
        const message = command.getOptionValueSource('message') === 'cli' ? opts.message : undefined
        const prId = parsePullRequestId(id)
        const mergeStrategy = parseMergeStrategy(strategy)
        const mergeMessage = resolveMergeMessage(message)
        const { workspace, repoSlug } = resolveWorkspaceRepo(opts)

        if (!confirmedByFlag) {
          const confirmed = await p.confirm({
            message: `Merge PR #${prId} into ${workspace}/${repoSlug}?`,
            initialValue: false,
          })

          if (p.isCancel(confirmed) || !confirmed) {
            logger.warn(`Merge cancelled. PR #${prId} was not merged.`)
            return
          }
        }

        const payload: PrMergePayload = {}
        if (mergeStrategy) payload.merge_strategy = mergeStrategy
        if (mergeMessage) payload.message = mergeMessage

        const client = await createBitbucketClient()
        const prService = new PrService(client)
        const pullRequest = await prService.merge(workspace, repoSlug, prId, payload)

        logger.success(`Merged PR #${pullRequest.id}: ${pullRequest.links.html.href}`)
      } catch (err) {
        if (err instanceof CliError) {
          throw err
        }

        if (err instanceof BitbucketApiError) {
          throw mapPullRequestApiError(err, id)
        }

        throw new CliError('An unexpected error occurred.', 2)
      } finally {
        command.setOptionValueWithSource('yes', false, 'default')
        command.setOptionValueWithSource('strategy', undefined, 'default')
        command.setOptionValueWithSource('message', undefined, 'default')
      }
    },
  )

prCommand
  .command('comments')
  .description('List comments on a pull request')
  .argument('<id>', 'Pull request ID')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .option('--json', 'Output raw JSON', false)
  .action(async (id: string, opts: { repo?: string; workspace?: string; json?: boolean }) => {
    try {
      const prId = parsePullRequestId(id)
      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const prService = new PrService(client)
      const comments = await prService.comments(workspace, repoSlug, prId)
      const wantsJson = opts.json === true
      opts.json = false

      if (wantsJson) {
        logger.log(formatJson(comments))
        return
      }

      if (comments.length === 0) {
        logger.log(`PR #${prId} has no comments.`)
        return
      }

      logger.log(
        formatTable(
          [
            { header: 'ID', key: 'id' },
            { header: 'Author', key: 'author' },
            { header: 'Created', key: 'created' },
            { header: 'Raw', key: 'raw' },
            { header: 'Rendered', key: 'rendered' },
          ],
          comments.map((comment) => {
            const content = formatCommentContent(comment)
            return {
              id: comment.id,
              author: comment.user?.display_name ?? 'Unknown',
              created: formatDate(comment.created_on),
              raw: content.raw,
              rendered: content.rendered,
            }
          }),
        ),
      )
    } catch (err) {
      if (err instanceof CliError) {
        throw err
      }

      if (err instanceof BitbucketApiError) {
        throw mapPullRequestApiError(err, id)
      }

      throw new CliError('An unexpected error occurred.', 2)
    }
  })

prCommand
  .command('comment')
  .description('Post a comment on a pull request')
  .argument('<id>', 'Pull request ID')
  .requiredOption('--message <text>', 'Comment message')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .action(async (id: string, opts: { message: string; repo?: string; workspace?: string }) => {
    try {
      const prId = parsePullRequestId(id)
      const message = resolveCommentMessage(opts.message)
      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const prService = new PrService(client)
      const comment = await prService.comment(workspace, repoSlug, prId, message)

      logger.success(`Comment #${comment.id} posted on PR #${prId}.`)
    } catch (err) {
      if (err instanceof CliError) {
        throw err
      }

      if (err instanceof BitbucketApiError) {
        throw mapPullRequestApiError(err, id)
      }

      throw new CliError('An unexpected error occurred.', 2)
    }
  })

prCommand
  .command('files')
  .description('List files changed in a pull request')
  .argument('<id>', 'Pull request ID')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .option('--json', 'Output raw JSON', false)
  .action(async (id: string, opts: { repo?: string; workspace?: string; json?: boolean }) => {
    try {
      const prId = parsePullRequestId(id)
      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const prService = new PrService(client)
      const files = await prService.files(workspace, repoSlug, prId)
      const wantsJson = opts.json === true
      opts.json = false

      if (wantsJson) {
        logger.log(formatJson(files))
        return
      }

      if (files.length === 0) {
        logger.log(`PR #${prId} has no changed files.`)
        return
      }

      logger.log(
        formatTable(
          [
            { header: 'Status', key: 'status' },
            { header: 'Type', key: 'type' },
            { header: 'Old Path', key: 'oldPath' },
            { header: 'Path', key: 'path' },
            { header: 'Added', key: 'added' },
            { header: 'Removed', key: 'removed' },
          ],
          files.map((file) => ({
            status: file.status ?? '',
            type: file.type,
            oldPath: formatOptionalDiffstatPath(file),
            path: formatDiffstatPath(file),
            added: formatLineCount(file.lines_added),
            removed: formatLineCount(file.lines_removed),
          })),
        ),
      )
    } catch (err) {
      if (err instanceof CliError) {
        throw err
      }

      if (err instanceof BitbucketApiError) {
        throw mapPullRequestApiError(err, id)
      }

      throw new CliError('An unexpected error occurred.', 2)
    }
  })

prCommand
  .command('diff')
  .description('Show the unified diff for a pull request')
  .argument('<id>', 'Pull request ID')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .action(async (id: string, opts: { repo?: string; workspace?: string }) => {
    try {
      const prId = parsePullRequestId(id)
      const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
      const client = await createBitbucketClient()
      const prService = new PrService(client)
      const diff = await prService.diff(workspace, repoSlug, prId)

      if (diff.trim().length === 0) {
        logger.log(`PR #${prId} has no diff changes.`)
        return
      }

      logger.log(diff)
    } catch (err) {
      if (err instanceof CliError) {
        throw err
      }

      if (err instanceof BitbucketApiError) {
        throw mapPullRequestApiError(err, id)
      }

      throw new CliError('An unexpected error occurred.', 2)
    }
  })

prCommand
  .command('update')
  .description('Update an existing pull request')
  .requiredOption('--id <number>', 'Pull request ID')
  .option('--title <title>', 'New pull request title')
  .option('--description <text>', 'New pull request description (markdown supported)')
  .option('--description-file <path>', 'Read description from a markdown file')
  .option('--target <branch>', 'New target branch')
  .option('--repo <slug>', 'Repository slug')
  .option('--workspace <name>', 'Bitbucket workspace name')
  .action(
    async (opts: {
      id: string
      title?: string
      description?: string
      descriptionFile?: string
      target?: string
      repo?: string
      workspace?: string
    }) => {
      try {
        const { workspace, repoSlug } = resolveWorkspaceRepo(opts)
        const client = await createBitbucketClient()
        const service = new PrService(client)

        const payload: PrUpdatePayload = {}
        if (opts.title) payload.title = opts.title
        const resolvedDesc = resolveDescription(opts.description, opts.descriptionFile)
        if (resolvedDesc) payload.description = resolvedDesc
        if (opts.target) payload.destination = { branch: { name: opts.target } }

        const pullRequest = await service.update(workspace, repoSlug, Number(opts.id), payload)
        logger.success(`PR #${pullRequest.id} updated: ${pullRequest.links.html.href}`)
      } catch (err) {
        if (err instanceof CliError) {
          throw err
        }

        if (err instanceof BitbucketApiError) {
          if (err.status === 404) {
            throw new CliError(`PR #${opts.id} not found.`, 1)
          }

          throw mapApiErrorToCliError(err)
        }

        throw new CliError('An unexpected error occurred.', 2)
      }
    },
  )
