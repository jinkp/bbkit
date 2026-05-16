import { describe, expect, it, vi } from 'vitest'
import { PrService, type PrCreatePayload, type PrMergePayload } from '../src/services/pr.service.js'
import type {
  BitbucketCommit,
  BitbucketCommitStatus,
  BitbucketPullRequest,
  BitbucketPullRequestComment,
  BitbucketPullRequestDiffstat,
  BitbucketPullRequestTask,
} from '../src/types/bitbucket.types.js'

async function* createPullRequestPages(items: BitbucketPullRequest[]) {
  for (const item of items) {
    yield item
  }
}

async function* createCommentPages(items: BitbucketPullRequestComment[]) {
  for (const item of items) {
    yield item
  }
}

async function* createDiffstatPages(items: BitbucketPullRequestDiffstat[]) {
  for (const item of items) {
    yield item
  }
}

async function* createCommitPages(items: BitbucketCommit[]) {
  for (const item of items) {
    yield item
  }
}

async function* createTaskPages(items: BitbucketPullRequestTask[]) {
  for (const item of items) {
    yield item
  }
}

async function* createCheckPages(items: BitbucketCommitStatus[]) {
  for (const item of items) {
    yield item
  }
}

describe('PrService', () => {
  it('lists open pull requests with pagination', async () => {
    const pullRequests: BitbucketPullRequest[] = [
      {
        id: 101,
        title: 'Add feature',
        state: 'OPEN',
        source: { branch: { name: 'feature/test' }, repository: { full_name: 'ws/repo' } },
        destination: { branch: { name: 'main' } },
        author: { display_name: 'Jane Doe' },
        created_on: '2026-04-28T00:00:00.000Z',
        updated_on: '2026-04-28T01:00:00.000Z',
        links: { html: { href: 'https://bitbucket.org/ws/repo/pull-requests/101' } },
      },
    ]

    const client = {
      get: vi.fn(),
      paginate: vi.fn(() => createPullRequestPages(pullRequests)),
      post: vi.fn(),
      patch: vi.fn(),
    }

    const service = new PrService(client)
    const result = await service.list('workspace', 'repo')

    expect(client.paginate).toHaveBeenCalledWith('/repositories/workspace/repo/pullrequests?state=OPEN')
    expect(result).toEqual(pullRequests)
  })

  it('lists open pull requests by default using only the OPEN state query parameter', async () => {
    const client = {
      get: vi.fn(),
      paginate: vi.fn(() => createPullRequestPages([])),
      post: vi.fn(),
      patch: vi.fn(),
    }

    const service = new PrService(client)
    await service.list('workspace', 'repo')

    const requestedPath = client.paginate.mock.calls[0][0]
    const url = new URL(requestedPath, 'https://api.bitbucket.org/2.0')

    expect(url.pathname).toBe('/repositories/workspace/repo/pullrequests')
    expect(url.searchParams.getAll('state')).toEqual(['OPEN'])
    expect(url.searchParams.has('q')).toBe(false)
  })

  it('lists pull requests with repeated deduped states and expands all states explicitly', async () => {
    const client = {
      get: vi.fn(),
      paginate: vi.fn(() => createPullRequestPages([])),
      post: vi.fn(),
      patch: vi.fn(),
    }
    const service = new PrService(client)

    await service.list('workspace', 'repo', { states: ['OPEN', 'DECLINED', 'OPEN'] })
    await service.list('workspace', 'repo', { states: ['OPEN', 'MERGED', 'DECLINED', 'SUPERSEDED'] })

    const repeatedUrl = new URL(client.paginate.mock.calls[0][0], 'https://api.bitbucket.org/2.0')
    const allUrl = new URL(client.paginate.mock.calls[1][0], 'https://api.bitbucket.org/2.0')

    expect(repeatedUrl.searchParams.getAll('state')).toEqual(['OPEN', 'DECLINED'])
    expect(allUrl.searchParams.getAll('state')).toEqual([
      'OPEN',
      'MERGED',
      'DECLINED',
      'SUPERSEDED',
    ])
  })

  it('combines author, reviewer, source, and target filters into one Bitbucket query', async () => {
    const client = {
      get: vi.fn(),
      paginate: vi.fn(() => createPullRequestPages([])),
      post: vi.fn(),
      patch: vi.fn(),
    }
    const service = new PrService(client)

    await service.list('workspace', 'repo', {
      author: '{author-uuid}',
      reviewer: '{reviewer-uuid}',
      source: 'feature/pr-list',
      target: 'main',
    })

    const url = new URL(client.paginate.mock.calls[0][0], 'https://api.bitbucket.org/2.0')

    expect(url.searchParams.getAll('state')).toEqual(['OPEN'])
    expect(url.searchParams.get('q')).toBe(
      'author.uuid = "{author-uuid}" AND reviewers.uuid = "{reviewer-uuid}" AND source.branch.name = "feature/pr-list" AND destination.branch.name = "main"',
    )
  })

  it('escapes query string literals and URL-encodes the combined query exactly once', async () => {
    const client = {
      get: vi.fn(),
      paginate: vi.fn(() => createPullRequestPages([])),
      post: vi.fn(),
      patch: vi.fn(),
    }
    const service = new PrService(client)

    await service.list('workspace', 'repo', {
      author: '{abc def}',
      source: 'feature/quote"and\\slash',
      target: 'release candidate',
    })

    const requestedPath = client.paginate.mock.calls[0][0]
    const url = new URL(requestedPath, 'https://api.bitbucket.org/2.0')
    const expectedQuery =
      'author.uuid = "{abc def}" AND source.branch.name = "feature/quote\\"and\\\\slash" AND destination.branch.name = "release candidate"'
    const expectedSearch = new URLSearchParams()
    expectedSearch.append('state', 'OPEN')
    expectedSearch.set('q', expectedQuery)

    expect(url.searchParams.get('q')).toBe(expectedQuery)
    expect(requestedPath).toBe(`/repositories/workspace/repo/pullrequests?${expectedSearch.toString()}`)
  })

  it('creates a pull request with the expected payload', async () => {
    const payload: PrCreatePayload = {
      title: 'Add feature',
      source: { branch: { name: 'feature/test' } },
      destination: { branch: { name: 'main' } },
      draft: true,
    }
    const pullRequest: BitbucketPullRequest = {
      id: 101,
      title: 'Add feature',
      state: 'OPEN',
      source: { branch: { name: 'feature/test' }, repository: { full_name: 'ws/repo' } },
      destination: { branch: { name: 'main' } },
      author: { display_name: 'Jane Doe' },
      created_on: '2026-04-28T00:00:00.000Z',
      updated_on: '2026-04-28T01:00:00.000Z',
      links: { html: { href: 'https://bitbucket.org/ws/repo/pull-requests/101' } },
    }

    const client = {
      get: vi.fn(),
      paginate: vi.fn(),
      post: vi.fn().mockResolvedValue(pullRequest),
      patch: vi.fn(),
    }

    const service = new PrService(client)
    const result = await service.create('workspace', 'repo', payload)

    expect(client.post).toHaveBeenCalledWith('/repositories/workspace/repo/pullrequests', payload)
    expect(result).toEqual(pullRequest)
  })

  it('fetches a pull request by id from the repository endpoint', async () => {
    const pullRequest: BitbucketPullRequest = {
      id: 77,
      title: 'Checkout this branch',
      state: 'OPEN',
      source: { branch: { name: 'feature/checkout-pr' }, repository: { full_name: 'ws/repo' } },
      destination: { branch: { name: 'main' } },
      author: { display_name: 'Jane Doe' },
      created_on: '2026-04-28T00:00:00.000Z',
      updated_on: '2026-04-28T01:00:00.000Z',
      links: { html: { href: 'https://bitbucket.org/ws/repo/pull-requests/77' } },
    }
    const client = {
      get: vi.fn().mockResolvedValue(pullRequest),
      paginate: vi.fn(),
      post: vi.fn(),
      patch: vi.fn(),
    }

    const service = new PrService(client)
    const result = await service.get('workspace', 'repo', 77)

    expect(client.get).toHaveBeenCalledWith('/repositories/workspace/repo/pullrequests/77')
    expect(result.source.branch.name).toBe('feature/checkout-pr')
  })

  it('approves a pull request through the Bitbucket approval endpoint', async () => {
    const client = {
      get: vi.fn(),
      paginate: vi.fn(),
      post: vi.fn().mockResolvedValue({ approved: true }),
      patch: vi.fn(),
    }

    const service = new PrService(client)

    await service.approve('workspace', 'repo', 123)

    expect(client.post).toHaveBeenCalledWith(
      '/repositories/workspace/repo/pullrequests/123/approve',
      {},
    )
  })

  it('adds a decline reason comment before declining the pull request', async () => {
    const client = {
      get: vi.fn(),
      paginate: vi.fn(),
      post: vi.fn().mockResolvedValue({ state: 'DECLINED' }),
      patch: vi.fn(),
    }

    const service = new PrService(client)

    await service.decline('workspace', 'repo', 123, 'Faltan pruebas')

    expect(client.post).toHaveBeenNthCalledWith(
      1,
      '/repositories/workspace/repo/pullrequests/123/comments',
      { content: { raw: 'Faltan pruebas' } },
    )
    expect(client.post).toHaveBeenNthCalledWith(
      2,
      '/repositories/workspace/repo/pullrequests/123/decline',
      {},
    )
  })

  it('merges a pull request through the Bitbucket merge endpoint with the required type', async () => {
    const pullRequest: BitbucketPullRequest = {
      id: 123,
      title: 'Ready to merge',
      state: 'MERGED',
      source: { branch: { name: 'feature/merge' }, repository: { full_name: 'ws/repo' } },
      destination: { branch: { name: 'main' } },
      author: { display_name: 'Jane Doe' },
      created_on: '2026-05-03T00:00:00.000Z',
      updated_on: '2026-05-03T01:00:00.000Z',
      links: { html: { href: 'https://bitbucket.org/ws/repo/pull-requests/123' } },
    }
    const client = {
      get: vi.fn(),
      paginate: vi.fn(),
      post: vi.fn().mockResolvedValue(pullRequest),
      patch: vi.fn(),
    }

    const service = new PrService(client)
    const result = await service.merge('workspace', 'repo', 123, {})

    expect(client.post).toHaveBeenCalledWith(
      '/repositories/workspace/repo/pullrequests/123/merge',
      { type: 'pullrequest_merge_parameters' },
    )
    expect(result).toEqual(pullRequest)
  })

  it('sends verified merge strategy and message fields when merging a pull request', async () => {
    const payload: PrMergePayload = {
      message: 'Merge feature X',
      merge_strategy: 'squash',
    }
    const pullRequest: BitbucketPullRequest = {
      id: 124,
      title: 'Squash me',
      state: 'MERGED',
      source: { branch: { name: 'feature/squash' }, repository: { full_name: 'ws/repo' } },
      destination: { branch: { name: 'main' } },
      author: { display_name: 'Jane Doe' },
      created_on: '2026-05-03T00:00:00.000Z',
      updated_on: '2026-05-03T01:00:00.000Z',
      links: { html: { href: 'https://bitbucket.org/ws/repo/pull-requests/124' } },
    }
    const client = {
      get: vi.fn(),
      paginate: vi.fn(),
      post: vi.fn().mockResolvedValue(pullRequest),
      patch: vi.fn(),
    }

    const service = new PrService(client)
    const result = await service.merge('workspace', 'repo', 124, payload)

    expect(client.post).toHaveBeenCalledWith(
      '/repositories/workspace/repo/pullrequests/124/merge',
      {
        type: 'pullrequest_merge_parameters',
        message: 'Merge feature X',
        merge_strategy: 'squash',
      },
    )
    expect(result).toEqual(pullRequest)
  })

  it('lists pull request comments from the Bitbucket comments endpoint with pagination', async () => {
    const comments: BitbucketPullRequestComment[] = [
      {
        id: 9001,
        created_on: '2026-05-01T12:00:00.000Z',
        user: { display_name: 'Jane Reviewer' },
        content: {
          raw: 'Please add tests.',
          markup: 'markdown',
          html: '<p>Please add tests.</p>',
        },
      },
      {
        id: 9002,
        created_on: '2026-05-01T13:00:00.000Z',
        user: { display_name: 'John Reviewer' },
        content: {
          raw: 'Looks good now.',
          markup: 'markdown',
          html: '<p>Looks good now.</p>',
        },
      },
    ]
    const client = {
      get: vi.fn(),
      paginate: vi.fn(() => createCommentPages(comments)),
      post: vi.fn(),
      patch: vi.fn(),
    }

    const service = new PrService(client)
    const result = await service.comments('workspace', 'repo', 123)

    expect(client.paginate).toHaveBeenCalledWith(
      '/repositories/workspace/repo/pullrequests/123/comments',
    )
    expect(result).toEqual(comments)
  })

  it('posts a pull request comment and returns the created comment', async () => {
    const comment: BitbucketPullRequestComment = {
      id: 9003,
      created_on: '2026-05-02T10:00:00.000Z',
      user: { display_name: 'Jane Reviewer' },
      content: {
        raw: 'Revisa este caso borde',
        markup: 'markdown',
        html: '<p>Revisa este caso borde</p>',
      },
    }
    const client = {
      get: vi.fn(),
      paginate: vi.fn(),
      post: vi.fn().mockResolvedValue(comment),
      patch: vi.fn(),
    }

    const service = new PrService(client)
    const result = await service.comment('workspace', 'repo', 123, 'Revisa este caso borde')

    expect(client.post).toHaveBeenCalledWith(
      '/repositories/workspace/repo/pullrequests/123/comments',
      { content: { raw: 'Revisa este caso borde' } },
    )
    expect(result).toEqual(comment)
  })

  it('lists pull request changed files from the Bitbucket diffstat endpoint with pagination', async () => {
    const files: BitbucketPullRequestDiffstat[] = [
      {
        type: 'diffstat',
        status: 'modified',
        lines_added: 12,
        lines_removed: 3,
        old: { path: 'src/old-name.ts', type: 'commit_file' },
        new: { path: 'src/new-name.ts', type: 'commit_file' },
      },
      {
        type: 'diffstat',
        status: 'added',
        lines_added: 20,
        lines_removed: 0,
        new: { path: 'tests/new-file.test.ts', type: 'commit_file' },
      },
    ]
    const client = {
      get: vi.fn(),
      paginate: vi.fn(() => createDiffstatPages(files)),
      post: vi.fn(),
      patch: vi.fn(),
    }

    const service = new PrService(client)
    const result = await service.files('workspace', 'repo', 123)

    expect(client.paginate).toHaveBeenCalledWith(
      '/repositories/workspace/repo/pullrequests/123/diffstat',
    )
    expect(result).toEqual(files)
  })

  it('fetches a pull request unified diff from the Bitbucket diff endpoint', async () => {
    const diff = 'diff --git a/src/app.ts b/src/app.ts\n+console.log("new")\n'
    const client = {
      get: vi.fn(),
      getText: vi.fn().mockResolvedValue(diff),
      paginate: vi.fn(),
      post: vi.fn(),
      patch: vi.fn(),
    }

    const service = new PrService(client)
    const result = await service.diff('workspace', 'repo', 123)

    expect(client.getText).toHaveBeenCalledWith('/repositories/workspace/repo/pullrequests/123/diff')
    expect(result).toBe(diff)
  })

  it('lists pull request commits from the Bitbucket commits endpoint with pagination', async () => {
    const commits: BitbucketCommit[] = [
      {
        hash: 'abcdef123456',
        message: 'Add workflow command',
        date: '2026-05-10T12:00:00.000Z',
        author: { raw: 'Jane Doe <jane@example.com>', user: { display_name: 'Jane Doe' } },
        links: { html: { href: 'https://bitbucket.org/ws/repo/commits/abcdef123456' } },
      },
    ]
    const client = {
      get: vi.fn(),
      paginate: vi.fn(() => createCommitPages(commits)),
      post: vi.fn(),
      patch: vi.fn(),
    }

    const service = new PrService(client)
    const result = await service.commits('workspace', 'repo', 123)

    expect(client.paginate).toHaveBeenCalledWith('/repositories/workspace/repo/pullrequests/123/commits')
    expect(result).toEqual(commits)
  })

  it('lists pull request tasks from the Bitbucket tasks endpoint with pagination', async () => {
    const tasks: BitbucketPullRequestTask[] = [
      {
        id: 501,
        state: 'OPEN',
        content: { raw: 'Add tests' },
        created_on: '2026-05-10T13:00:00.000Z',
        creator: { display_name: 'Jane Reviewer' },
      },
    ]
    const client = {
      get: vi.fn(),
      paginate: vi.fn(() => createTaskPages(tasks)),
      post: vi.fn(),
      patch: vi.fn(),
    }

    const service = new PrService(client)
    const result = await service.tasks('workspace', 'repo', 123)

    expect(client.paginate).toHaveBeenCalledWith('/repositories/workspace/repo/pullrequests/123/tasks')
    expect(result).toEqual(tasks)
  })

  it('lists pull request checks from the Bitbucket statuses endpoint with pagination', async () => {
    const checks: BitbucketCommitStatus[] = [
      {
        key: 'ci',
        name: 'CI',
        state: 'SUCCESSFUL',
        updated_on: '2026-05-10T14:00:00.000Z',
        url: 'https://ci.example/build/1',
      },
    ]
    const client = {
      get: vi.fn(),
      paginate: vi.fn(() => createCheckPages(checks)),
      post: vi.fn(),
      patch: vi.fn(),
    }

    const service = new PrService(client)
    const result = await service.checks('workspace', 'repo', 123)

    expect(client.paginate).toHaveBeenCalledWith('/repositories/workspace/repo/pullrequests/123/statuses')
    expect(result).toEqual(checks)
  })

  it('creates a pull request task with the Bitbucket raw-content payload', async () => {
    const task: BitbucketPullRequestTask = {
      id: 502,
      state: 'OPEN',
      content: { raw: 'Review auth flow' },
      created_on: '2026-05-10T15:00:00.000Z',
    }
    const client = {
      get: vi.fn(),
      paginate: vi.fn(),
      post: vi.fn().mockResolvedValue(task),
      patch: vi.fn(),
    }

    const service = new PrService(client)
    const result = await service.createTask('workspace', 'repo', 123, 'Review auth flow')

    expect(client.post).toHaveBeenCalledWith('/repositories/workspace/repo/pullrequests/123/tasks', {
      content: { raw: 'Review auth flow' },
    })
    expect(result).toEqual(task)
  })

  it('resolves a pull request task through client.patch using the resolved state payload', async () => {
    const task: BitbucketPullRequestTask = {
      id: 502,
      state: 'RESOLVED',
      content: { raw: 'Review auth flow' },
      updated_on: '2026-05-10T16:00:00.000Z',
    }
    const client = {
      get: vi.fn(),
      paginate: vi.fn(),
      post: vi.fn(),
      patch: vi.fn().mockResolvedValue(task),
    }

    const service = new PrService(client)
    const result = await service.resolveTask('workspace', 'repo', 123, 502)

    expect(client.patch).toHaveBeenCalledWith('/repositories/workspace/repo/pullrequests/123/tasks/502', {
      state: 'RESOLVED',
    })
    expect(result).toEqual(task)
  })
})
