import type { BitbucketClient } from '../client/bitbucket.client.js'
import type {
  BitbucketCommit,
  BitbucketCommitStatus,
  BitbucketPullRequest,
  BitbucketPullRequestComment,
  BitbucketPullRequestDiffstat,
  BitbucketPullRequestState,
  BitbucketPullRequestTask,
} from '../types/bitbucket.types.js'

export interface PrCreatePayload {
  title: string
  source: { branch: { name: string } }
  destination: { branch: { name: string } }
  draft?: boolean
  description?: string
}

export interface PrUpdatePayload {
  title?: string
  description?: string
  destination?: { branch: { name: string } }
}

export interface PrListOptions {
  states?: BitbucketPullRequestState[]
  author?: string
  reviewer?: string
  source?: string
  target?: string
}

export type PrMergeStrategy =
  | 'merge_commit'
  | 'squash'
  | 'fast_forward'
  | 'squash_fast_forward'
  | 'rebase_fast_forward'
  | 'rebase_merge'

export interface PrMergePayload {
  message?: string
  merge_strategy?: PrMergeStrategy
}

interface BitbucketPrMergePayload extends PrMergePayload {
  type: 'pullrequest_merge_parameters'
}

type PrServiceClient = Pick<BitbucketClient, 'get' | 'paginate' | 'post' | 'patch'> & {
  getText?: (path: string) => Promise<string>
}

const DEFAULT_PR_LIST_STATES: readonly BitbucketPullRequestState[] = ['OPEN']

function dedupeStates(states: readonly BitbucketPullRequestState[]): BitbucketPullRequestState[] {
  return [...new Set(states)]
}

function escapeQueryLiteral(value: string): string {
  return value.replace(/\\/g, '\\\\').replace(/"/g, '\\"')
}

function createQueryExpression(options: PrListOptions): string | undefined {
  const expressions = [
    options.author ? `author.uuid = "${escapeQueryLiteral(options.author)}"` : undefined,
    options.reviewer ? `reviewers.uuid = "${escapeQueryLiteral(options.reviewer)}"` : undefined,
    options.source ? `source.branch.name = "${escapeQueryLiteral(options.source)}"` : undefined,
    options.target ? `destination.branch.name = "${escapeQueryLiteral(options.target)}"` : undefined,
  ].filter((expression): expression is string => expression !== undefined)

  return expressions.length > 0 ? expressions.join(' AND ') : undefined
}

function createPullRequestListPath(workspace: string, repoSlug: string, options: PrListOptions): string {
  const params = new URLSearchParams()
  const states = dedupeStates(options.states?.length ? options.states : DEFAULT_PR_LIST_STATES)
  const queryExpression = createQueryExpression(options)

  for (const state of states) {
    params.append('state', state)
  }

  if (queryExpression) {
    params.set('q', queryExpression)
  }

  return `/repositories/${workspace}/${repoSlug}/pullrequests?${params.toString()}`
}

export class PrService {
  constructor(private readonly client: PrServiceClient) {}

  get(workspace: string, repoSlug: string, prId: number): Promise<BitbucketPullRequest> {
    return this.client.get<BitbucketPullRequest>(
      `/repositories/${workspace}/${repoSlug}/pullrequests/${prId}`,
    )
  }

  async list(
    workspace: string,
    repoSlug: string,
    options: PrListOptions = {},
  ): Promise<BitbucketPullRequest[]> {
    const pullRequests: BitbucketPullRequest[] = []

    for await (const pullRequest of this.client.paginate<BitbucketPullRequest>(
      createPullRequestListPath(workspace, repoSlug, options),
    )) {
      pullRequests.push(pullRequest)
    }

    return pullRequests
  }

  async comments(
    workspace: string,
    repoSlug: string,
    prId: number,
  ): Promise<BitbucketPullRequestComment[]> {
    const comments: BitbucketPullRequestComment[] = []

    for await (const comment of this.client.paginate<BitbucketPullRequestComment>(
      `/repositories/${workspace}/${repoSlug}/pullrequests/${prId}/comments`,
    )) {
      comments.push(comment)
    }

    return comments
  }

  async files(
    workspace: string,
    repoSlug: string,
    prId: number,
  ): Promise<BitbucketPullRequestDiffstat[]> {
    const files: BitbucketPullRequestDiffstat[] = []

    for await (const file of this.client.paginate<BitbucketPullRequestDiffstat>(
      `/repositories/${workspace}/${repoSlug}/pullrequests/${prId}/diffstat`,
    )) {
      files.push(file)
    }

    return files
  }

  async commits(workspace: string, repoSlug: string, prId: number): Promise<BitbucketCommit[]> {
    const commits: BitbucketCommit[] = []

    for await (const commit of this.client.paginate<BitbucketCommit>(
      `/repositories/${workspace}/${repoSlug}/pullrequests/${prId}/commits`,
    )) {
      commits.push(commit)
    }

    return commits
  }

  async tasks(
    workspace: string,
    repoSlug: string,
    prId: number,
  ): Promise<BitbucketPullRequestTask[]> {
    const tasks: BitbucketPullRequestTask[] = []

    for await (const task of this.client.paginate<BitbucketPullRequestTask>(
      `/repositories/${workspace}/${repoSlug}/pullrequests/${prId}/tasks`,
    )) {
      tasks.push(task)
    }

    return tasks
  }

  createTask(
    workspace: string,
    repoSlug: string,
    prId: number,
    message: string,
  ): Promise<BitbucketPullRequestTask> {
    return this.client.post<BitbucketPullRequestTask>(
      `/repositories/${workspace}/${repoSlug}/pullrequests/${prId}/tasks`,
      { content: { raw: message } },
    )
  }

  resolveTask(
    workspace: string,
    repoSlug: string,
    prId: number,
    taskId: number,
  ): Promise<BitbucketPullRequestTask> {
    return this.client.patch<BitbucketPullRequestTask>(
      `/repositories/${workspace}/${repoSlug}/pullrequests/${prId}/tasks/${taskId}`,
      { state: 'RESOLVED' },
    )
  }

  async checks(workspace: string, repoSlug: string, prId: number): Promise<BitbucketCommitStatus[]> {
    const checks: BitbucketCommitStatus[] = []

    for await (const check of this.client.paginate<BitbucketCommitStatus>(
      `/repositories/${workspace}/${repoSlug}/pullrequests/${prId}/statuses`,
    )) {
      checks.push(check)
    }

    return checks
  }

  diff(workspace: string, repoSlug: string, prId: number): Promise<string> {
    if (!this.client.getText) {
      throw new Error('Bitbucket client does not support text responses.')
    }

    return this.client.getText(`/repositories/${workspace}/${repoSlug}/pullrequests/${prId}/diff`)
  }

  create(
    workspace: string,
    repoSlug: string,
    payload: PrCreatePayload,
  ): Promise<BitbucketPullRequest> {
    return this.client.post<BitbucketPullRequest>(
      `/repositories/${workspace}/${repoSlug}/pullrequests`,
      payload,
    )
  }

  update(
    workspace: string,
    repoSlug: string,
    prId: number,
    payload: PrUpdatePayload,
  ): Promise<BitbucketPullRequest> {
    return this.client.patch<BitbucketPullRequest>(
      `/repositories/${workspace}/${repoSlug}/pullrequests/${prId}`,
      payload,
    )
  }

  async approve(workspace: string, repoSlug: string, prId: number): Promise<void> {
    await this.client.post<unknown>(
      `/repositories/${workspace}/${repoSlug}/pullrequests/${prId}/approve`,
      {},
    )
  }

  async decline(workspace: string, repoSlug: string, prId: number, reason: string): Promise<void> {
    await this.comment(workspace, repoSlug, prId, reason)
    await this.client.post<unknown>(
      `/repositories/${workspace}/${repoSlug}/pullrequests/${prId}/decline`,
      {},
    )
  }

  comment(
    workspace: string,
    repoSlug: string,
    prId: number,
    message: string,
  ): Promise<BitbucketPullRequestComment> {
    return this.client.post<BitbucketPullRequestComment>(
      `/repositories/${workspace}/${repoSlug}/pullrequests/${prId}/comments`,
      { content: { raw: message } },
    )
  }

  merge(
    workspace: string,
    repoSlug: string,
    prId: number,
    payload: PrMergePayload,
  ): Promise<BitbucketPullRequest> {
    const bitbucketPayload: BitbucketPrMergePayload = {
      type: 'pullrequest_merge_parameters',
      ...payload,
    }

    return this.client.post<BitbucketPullRequest>(
      `/repositories/${workspace}/${repoSlug}/pullrequests/${prId}/merge`,
      bitbucketPayload,
    )
  }
}
