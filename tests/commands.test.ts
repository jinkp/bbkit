import { beforeEach, describe, expect, it, vi } from 'vitest'
import { Command } from 'commander'
import { BitbucketApiError } from '../src/errors/cli.errors.js'

const mocks = vi.hoisted(() => ({
  auth: {
    login: vi.fn(),
    status: vi.fn(),
    logout: vi.fn(),
    getCredentialsOrFail: vi.fn(),
  },
  repoList: vi.fn(),
  prList: vi.fn(),
  prCreate: vi.fn(),
  prGet: vi.fn(),
  prApprove: vi.fn(),
  prDecline: vi.fn(),
  prComments: vi.fn(),
  prComment: vi.fn(),
  prCommits: vi.fn(),
  prTasks: vi.fn(),
  prCreateTask: vi.fn(),
  prResolveTask: vi.fn(),
  prChecks: vi.fn(),
  prFiles: vi.fn(),
  prDiff: vi.fn(),
  prMerge: vi.fn(),
  clackConfirm: vi.fn(),
  browserOpen: vi.fn(),
  branchList: vi.fn(),
  branchStale: vi.fn(),
  pipelineList: vi.fn(),
  pipelineRun: vi.fn(),
  gitCheckout: vi.fn(),
}))

vi.mock('../src/services/auth.service.js', () => ({
  authService: mocks.auth,
}))

vi.mock('../src/services/repo.service.js', () => ({
  RepoService: vi.fn().mockImplementation(() => ({
    list: mocks.repoList,
  })),
}))

vi.mock('../src/services/pr.service.js', () => ({
  PrService: vi.fn().mockImplementation(() => ({
    list: mocks.prList,
    create: mocks.prCreate,
    get: mocks.prGet,
    approve: mocks.prApprove,
    decline: mocks.prDecline,
    comments: mocks.prComments,
    comment: mocks.prComment,
    commits: mocks.prCommits,
    tasks: mocks.prTasks,
    createTask: mocks.prCreateTask,
    resolveTask: mocks.prResolveTask,
    checks: mocks.prChecks,
    files: mocks.prFiles,
    diff: mocks.prDiff,
    merge: mocks.prMerge,
  })),
}))

vi.mock('@clack/prompts', () => ({
  confirm: mocks.clackConfirm,
  isCancel: (v: unknown) => v === Symbol.for('cancel'),
  group: vi.fn(),
  text: vi.fn(),
  password: vi.fn(),
  select: vi.fn(),
  spinner: vi.fn(() => ({ start: vi.fn(), stop: vi.fn() })),
  cancel: vi.fn(),
  log: { error: vi.fn(), info: vi.fn(), warn: vi.fn() },
}))

vi.mock('../src/services/git-checkout.service.js', () => ({
  GitCheckoutService: vi.fn().mockImplementation(() => ({
    checkoutSourceBranch: mocks.gitCheckout,
  })),
}))

vi.mock('../src/services/browser-open.service.js', () => ({
  BrowserOpenService: vi.fn().mockImplementation(() => ({
    openUrl: mocks.browserOpen,
  })),
}))

vi.mock('../src/services/branch.service.js', () => ({
  BranchService: vi.fn().mockImplementation(() => ({
    list: mocks.branchList,
    stale: mocks.branchStale,
  })),
}))

vi.mock('../src/services/pipeline.service.js', () => ({
  PipelineService: vi.fn().mockImplementation(() => ({
    list: mocks.pipelineList,
    run: mocks.pipelineRun,
  })),
}))

vi.mock('../src/config/config.store.js', () => ({
  configStore: {
    getWorkspace: vi.fn().mockReturnValue('testworkspace'),
    getUsername: vi.fn().mockReturnValue('testuser'),
    setUsername: vi.fn(),
    setWorkspace: vi.fn(),
    getDefaultOutput: vi.fn().mockReturnValue('table'),
    clear: vi.fn(),
  },
}))

vi.mock('../src/utils/git-remote.js', () => ({
  inferFromGitRemote: vi.fn().mockReturnValue({ workspace: 'testworkspace', repoSlug: 'test-repo' }),
}))

vi.mock('../src/client/bitbucket.client.js', () => ({
  BitbucketClient: vi.fn().mockImplementation(() => ({})),
}))

import { authCommand } from '../src/commands/auth.command.js'
import { branchCommand } from '../src/commands/branch.command.js'
import { pipelineCommand } from '../src/commands/pipeline.command.js'
import { prCommand } from '../src/commands/pr.command.js'
import { repoCommand } from '../src/commands/repo.command.js'
import { createCliProgram } from '../src/program.js'
import { authService } from '../src/services/auth.service.js'

function createProgram(command: Command): Command {
  const program = new Command()
  program.exitOverride()
  program.addCommand(command)
  return program
}

async function runRootProgram(args: string[]): Promise<{ stdout: string; stderr: string }> {
  let stdout = ''
  let stderr = ''
  const program = createCliProgram({
    name: 'bbkit-cli-test',
    version: '9.8.7',
  })
  program.exitOverride()
  program.configureOutput({
    writeOut: (value) => {
      stdout += value
    },
    writeErr: (value) => {
      stderr += value
    },
  })

  try {
    await program.parseAsync(['node', 'bbk', ...args])
  } catch (error) {
    if (!(error instanceof Error) || error.constructor.name !== 'CommanderError') {
      throw error
    }
  }

  return { stdout, stderr }
}

describe('command layer', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    mocks.auth.getCredentialsOrFail.mockResolvedValue({ username: 'testuser', apiToken: 'token' })
    mocks.auth.status.mockResolvedValue({ authenticated: true, username: 'testuser', source: 'keychain' })
    mocks.auth.logout.mockResolvedValue(undefined)
    mocks.repoList.mockResolvedValue([
      {
        name: 'my-repo',
        description: 'A repo',
        language: 'TypeScript',
        updated_on: '2024-01-15T10:00:00Z',
        slug: 'my-repo',
        full_name: 'ws/my-repo',
        is_private: false,
      },
    ])
    mocks.prList.mockResolvedValue([])
    mocks.prCreate.mockResolvedValue({
      id: 1,
      title: 'Test PR',
      links: { html: { href: 'https://bitbucket.org/ws/repo/pull-requests/1' } },
    })
    mocks.prGet.mockResolvedValue({
      id: 123,
      title: 'Workflow PR',
      state: 'OPEN',
      source: { branch: { name: 'feature/checkout-pr' }, repository: { full_name: 'ws/repo' } },
      destination: { branch: { name: 'main' } },
      author: { display_name: 'Jane Doe' },
      created_on: '2026-04-28T00:00:00.000Z',
      updated_on: '2026-04-28T01:00:00.000Z',
      comment_count: 4,
      task_count: 2,
      reviewers: [{ display_name: 'Jane Reviewer', uuid: '{reviewer}' }],
      participants: [{ user: { display_name: 'John Approver' }, approved: true, state: 'approved' }],
      links: { html: { href: 'https://bitbucket.org/ws/repo/pull-requests/123' } },
    })
    mocks.gitCheckout.mockReturnValue({ branchName: 'feature/checkout-pr' })
    mocks.browserOpen.mockReturnValue({ url: 'https://bitbucket.org/ws/repo/pull-requests/123' })
    mocks.prApprove.mockResolvedValue(undefined)
    mocks.prDecline.mockResolvedValue(undefined)
    mocks.prComments.mockResolvedValue([
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
    ])
    mocks.prComment.mockResolvedValue({
      id: 9003,
      created_on: '2026-05-02T10:00:00.000Z',
      user: { display_name: 'Jane Reviewer' },
      content: {
        raw: 'Revisa este caso borde',
        markup: 'markdown',
        html: '<p>Revisa este caso borde</p>',
      },
    })
    mocks.prCommits.mockResolvedValue([
      {
        hash: 'abcdef1234567890',
        message: 'Add workflow commands\n\nBody',
        date: '2026-05-10T12:00:00.000Z',
        author: { raw: 'Jane Doe <jane@example.com>', user: { display_name: 'Jane Doe' } },
        links: { html: { href: 'https://bitbucket.org/ws/repo/commits/abcdef1234567890' } },
      },
    ])
    mocks.prTasks.mockResolvedValue([
      {
        id: 501,
        state: 'OPEN',
        content: { raw: 'Add tests' },
        created_on: '2026-05-10T13:00:00.000Z',
        creator: { display_name: 'Jane Reviewer' },
        assignee: { display_name: 'Dev Owner' },
      },
    ])
    mocks.prCreateTask.mockResolvedValue({
      id: 502,
      state: 'OPEN',
      content: { raw: 'Review auth flow' },
      created_on: '2026-05-10T15:00:00.000Z',
    })
    mocks.prResolveTask.mockResolvedValue({
      id: 502,
      state: 'RESOLVED',
      content: { raw: 'Review auth flow' },
      updated_on: '2026-05-10T16:00:00.000Z',
    })
    mocks.prChecks.mockResolvedValue([
      {
        key: 'ci',
        name: 'CI',
        state: 'SUCCESSFUL',
        description: 'Build passed',
        updated_on: '2026-05-10T14:00:00.000Z',
        url: 'https://ci.example/build/1',
      },
    ])
    mocks.prFiles.mockResolvedValue([
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
    ])
    mocks.prDiff.mockResolvedValue('diff --git a/src/app.ts b/src/app.ts\n+console.log("new")\n')
    mocks.prMerge.mockResolvedValue({
      id: 123,
      title: 'Merged PR',
      state: 'MERGED',
      source: { branch: { name: 'feature/merge' }, repository: { full_name: 'ws/repo' } },
      destination: { branch: { name: 'main' } },
      author: { display_name: 'Jane Doe' },
      created_on: '2026-05-03T00:00:00.000Z',
      updated_on: '2026-05-03T01:00:00.000Z',
      links: { html: { href: 'https://bitbucket.org/ws/repo/pull-requests/123' } },
    })
    mocks.clackConfirm.mockResolvedValue(true)
    mocks.branchList.mockResolvedValue([])
    mocks.branchStale.mockResolvedValue([])
    mocks.pipelineList.mockResolvedValue([])
    mocks.pipelineRun.mockResolvedValue({
      uuid: '{123}',
      build_number: 42,
      state: { name: 'PENDING' },
      target: { ref_name: 'develop', ref_type: 'branch' },
      created_on: '2024-01-15T10:00:00Z',
      links: { self: { href: 'https://bitbucket.org/ws/repo/pipelines/results/42' } },
    })
  })

  it('prints the package version with the standard --version option', async () => {
    const result = await runRootProgram(['--version'])

    expect(result.stdout).toBe('9.8.7\n')
    expect(result.stderr).toBe('')
  })

  it('prints the package version with the standard -V option', async () => {
    const result = await runRootProgram(['-V'])

    expect(result.stdout).toBe('9.8.7\n')
    expect(result.stderr).toBe('')
  })

  it('prints the package version with the explicit version command', async () => {
    const result = await runRootProgram(['version'])

    expect(result.stdout).toBe('9.8.7\n')
    expect(result.stderr).toBe('')
  })

  it('prints version details as JSON when requested', async () => {
    const result = await runRootProgram(['version', '--json'])

    expect(JSON.parse(result.stdout)).toMatchObject({
      name: 'bbkit-cli-test',
      version: '9.8.7',
      node: process.version,
      platform: process.platform,
      arch: process.arch,
    })
    expect(result.stderr).toBe('')
  })

  it('auth status shows authenticated message when authenticated', async () => {
    vi.mocked(authService.status).mockResolvedValue({
      authenticated: true,
      username: 'testuser',
      source: 'keychain',
    })

    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(authCommand).parseAsync(['node', 'bbk', 'auth', 'status'])

    expect(authService.status).toHaveBeenCalled()
    spy.mockRestore()
  })

  it('auth status handles unauthenticated state', async () => {
    vi.mocked(authService.status).mockResolvedValue({ authenticated: false })

    const spy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    await createProgram(authCommand).parseAsync(['node', 'bbk', 'auth', 'status'])

    expect(authService.status).toHaveBeenCalled()
    spy.mockRestore()
  })

  it('auth logout calls authService.logout', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(authCommand).parseAsync(['node', 'bbk', 'auth', 'logout'])

    expect(authService.logout).toHaveBeenCalled()
    spy.mockRestore()
  })

  it('repo list renders the spec table columns', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(repoCommand).parseAsync(['node', 'bbk', 'repo', 'list'])

    expect(mocks.repoList).toHaveBeenCalledWith('testworkspace')
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('Description'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('A repo'))
    expect(spy).not.toHaveBeenCalledWith(expect.stringContaining('Private'))
    spy.mockRestore()
  })

  it('repo list maps 404 errors with resource context', async () => {
    mocks.repoList.mockRejectedValue(
      new BitbucketApiError('Not Found', 404, 'https://api.bitbucket.org/2.0/repositories/testworkspace'),
    )

    await expect(
      createProgram(repoCommand).parseAsync(['node', 'bbk', 'repo', 'list']),
    ).rejects.toMatchObject({
      message: 'Resource not found: https://api.bitbucket.org/2.0/repositories/testworkspace.',
      exitCode: 1,
    })
  })

  it('pr list requests open pull requests by default', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'list'])

    expect(mocks.prList).toHaveBeenCalledWith('testworkspace', 'test-repo', { states: ['OPEN'] })
    spy.mockRestore()
  })

  it('pr list normalizes repeatable states and removes duplicates before calling the service', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync([
      'node',
      'bbk',
      'pr',
      'list',
      '--state',
      'merged',
      '--state',
      'OPEN',
      '--state',
      'merged',
      '--author',
      '{author-uuid}',
      '--reviewer',
      '{reviewer-uuid}',
      '--source',
      'feature/pr-list',
      '--target',
      'main',
    ])

    expect(mocks.prList).toHaveBeenCalledWith('testworkspace', 'test-repo', {
      states: ['MERGED', 'OPEN'],
      author: '{author-uuid}',
      reviewer: '{reviewer-uuid}',
      source: 'feature/pr-list',
      target: 'main',
    })
    spy.mockRestore()
  })

  it('pr list expands all states explicitly', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'list', '--all'])

    expect(mocks.prList).toHaveBeenCalledWith('testworkspace', 'test-repo', {
      states: ['OPEN', 'MERGED', 'DECLINED', 'SUPERSEDED'],
    })
    spy.mockRestore()
  })

  it('pr list rejects invalid states, conflicting all/state, and empty filters before calling the service', async () => {
    await expect(
      createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'list', '--state', 'closed']),
    ).rejects.toMatchObject({
      message: 'Invalid pull request state: closed. Supported states: OPEN, MERGED, DECLINED, SUPERSEDED.',
      exitCode: 1,
    })

    await expect(
      createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'list', '--all', '--state', 'OPEN']),
    ).rejects.toMatchObject({
      message: 'Use either --all or --state, not both.',
      exitCode: 1,
    })

    await expect(
      createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'list', '--source', '   ']),
    ).rejects.toMatchObject({
      message: 'Filter source cannot be empty.',
      exitCode: 1,
    })

    expect(mocks.prList).not.toHaveBeenCalled()
  })

  it('pr list prints raw JSON arrays and a friendly empty table state', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'list', '--json'])
    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'list'])

    expect(spy).toHaveBeenNthCalledWith(1, '[]')
    expect(spy).toHaveBeenNthCalledWith(2, 'No pull requests found for the selected filters.')
    spy.mockRestore()
  })

  it('pr create preserves source-branch validation message', async () => {
    mocks.prCreate.mockRejectedValue(new BitbucketApiError('Not Found', 404, '/branches/missing'))

    await expect(
      createProgram(prCommand).parseAsync([
        'node',
        'bbk',
        'pr',
        'create',
        '--source',
        'missing',
        '--target',
        'main',
        '--title',
        'Test PR',
      ]),
    ).rejects.toMatchObject({
      message: "Source branch 'missing' not found.",
      exitCode: 1,
    })
  })

  it('pipeline run outputs the pipeline id and url', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(pipelineCommand).parseAsync([
      'node',
      'bbk',
      'pipeline',
      'run',
      '--branch',
      'develop',
    ])

    expect(mocks.pipelineRun).toHaveBeenCalledWith('testworkspace', 'test-repo', 'develop')
    expect(spy).toHaveBeenNthCalledWith(1, expect.any(String), 'Pipeline #42 started')
    expect(spy).toHaveBeenNthCalledWith(2, 'URL: https://bitbucket.org/ws/repo/pipelines/results/42')
    spy.mockRestore()
  })

  it('pr checkout fetches the PR source branch and checks it out locally', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'checkout', '123'])

    expect(mocks.prGet).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(mocks.gitCheckout).toHaveBeenCalledWith('feature/checkout-pr')
    expect(spy).toHaveBeenCalledWith(
      expect.any(String),
      'Checked out PR #123 source branch: feature/checkout-pr',
    )
    spy.mockRestore()
  })

  it('pr checkout maps a missing pull request to a user-facing error', async () => {
    mocks.prGet.mockRejectedValue(new BitbucketApiError('Not Found', 404, '/pullrequests/404'))

    await expect(
      createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'checkout', '404']),
    ).rejects.toMatchObject({
      message: 'PR #404 not found.',
      exitCode: 1,
    })
  })

  it('pr open fetches the PR HTML URL and opens it in the default browser', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'open', '123'])

    expect(mocks.prGet).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(mocks.browserOpen).toHaveBeenCalledWith('https://bitbucket.org/ws/repo/pull-requests/123')
    expect(spy).toHaveBeenCalledWith(
      expect.any(String),
      'Opened PR #123: https://bitbucket.org/ws/repo/pull-requests/123',
    )
    spy.mockRestore()
  })

  it('pr open rejects invalid pull request ids before fetching', async () => {
    await expect(
      createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'open', 'abc']),
    ).rejects.toMatchObject({
      message: 'Invalid pull request ID: abc',
      exitCode: 1,
    })
    expect(mocks.prGet).not.toHaveBeenCalled()
    expect(mocks.browserOpen).not.toHaveBeenCalled()
  })

  it('pr open maps a missing pull request to a user-facing error', async () => {
    mocks.prGet.mockRejectedValue(new BitbucketApiError('Not Found', 404, '/pullrequests/404'))

    await expect(
      createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'open', '404']),
    ).rejects.toMatchObject({
      message: 'PR #404 not found.',
      exitCode: 1,
    })
  })

  it('pr approve approves the pull request and reports success', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'approve', '123'])

    expect(mocks.prApprove).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(spy).toHaveBeenCalledWith(expect.any(String), 'Approved PR #123.')
    spy.mockRestore()
  })

  it('pr approve rejects invalid pull request ids before approving', async () => {
    await expect(
      createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'approve', 'abc']),
    ).rejects.toMatchObject({
      message: 'Invalid pull request ID: abc',
      exitCode: 1,
    })

    expect(mocks.prApprove).not.toHaveBeenCalled()
  })

  it('pr approve maps a missing pull request to a user-facing error', async () => {
    mocks.prApprove.mockRejectedValue(new BitbucketApiError('Not Found', 404, '/pullrequests/404'))

    await expect(
      createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'approve', '404']),
    ).rejects.toMatchObject({
      message: 'PR #404 not found.',
      exitCode: 1,
    })
  })

  it('pr decline records the reason and declines the pull request', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync([
      'node',
      'bbk',
      'pr',
      'decline',
      '123',
      '--reason',
      'Faltan pruebas',
    ])

    expect(mocks.prDecline).toHaveBeenCalledWith('testworkspace', 'test-repo', 123, 'Faltan pruebas')
    expect(spy).toHaveBeenCalledWith(expect.any(String), 'Declined PR #123 with reason comment.')
    spy.mockRestore()
  })

  it('pr decline rejects invalid pull request ids before declining', async () => {
    await expect(
      createProgram(prCommand).parseAsync([
        'node',
        'bbk',
        'pr',
        'decline',
        'abc',
        '--reason',
        'Faltan pruebas',
      ]),
    ).rejects.toMatchObject({
      message: 'Invalid pull request ID: abc',
      exitCode: 1,
    })

    expect(mocks.prDecline).not.toHaveBeenCalled()
  })

  it('pr decline maps a missing pull request to a user-facing error', async () => {
    mocks.prDecline.mockRejectedValue(new BitbucketApiError('Not Found', 404, '/pullrequests/404'))

    await expect(
      createProgram(prCommand).parseAsync([
        'node',
        'bbk',
        'pr',
        'decline',
        '404',
        '--reason',
        'Faltan pruebas',
      ]),
    ).rejects.toMatchObject({
      message: 'PR #404 not found.',
      exitCode: 1,
    })
  })

  it('pr comments renders human-readable pull request comments', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'comments', '123'])

    expect(mocks.prComments).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('Jane Reviewer'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('Please add tests.'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('<p>Please add tests.</p>'))
    spy.mockRestore()
  })

  it('pr comments outputs raw JSON when requested', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'comments', '123', '--json'])

    expect(mocks.prComments).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('"id": 9001'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('"display_name": "Jane Reviewer"'))
    spy.mockRestore()
  })

  it('pr comments reports when a pull request has no comments', async () => {
    mocks.prComments.mockResolvedValue([])
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'comments', '123'])

    expect(mocks.prComments).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(spy).toHaveBeenCalledWith('PR #123 has no comments.')
    spy.mockRestore()
  })

  it('pr comments rejects invalid pull request ids before fetching comments', async () => {
    await expect(
      createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'comments', 'abc']),
    ).rejects.toMatchObject({
      message: 'Invalid pull request ID: abc',
      exitCode: 1,
    })

    expect(mocks.prComments).not.toHaveBeenCalled()
  })

  it('pr comments maps a missing pull request to a user-facing error', async () => {
    mocks.prComments.mockRejectedValue(new BitbucketApiError('Not Found', 404, '/pullrequests/404'))

    await expect(
      createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'comments', '404']),
    ).rejects.toMatchObject({
      message: 'PR #404 not found.',
      exitCode: 1,
    })
  })

  it('pr comment posts a message and reports the created comment id', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync([
      'node',
      'bbk',
      'pr',
      'comment',
      '123',
      '--message',
      'Revisa este caso borde',
    ])

    expect(mocks.prComment).toHaveBeenCalledWith(
      'testworkspace',
      'test-repo',
      123,
      'Revisa este caso borde',
    )
    expect(spy).toHaveBeenCalledWith(expect.any(String), 'Comment #9003 posted on PR #123.')
    spy.mockRestore()
  })

  it('pr comment rejects blank messages before posting', async () => {
    await expect(
      createProgram(prCommand).parseAsync([
        'node',
        'bbk',
        'pr',
        'comment',
        '123',
        '--message',
        '   ',
      ]),
    ).rejects.toMatchObject({
      message: 'Comment message cannot be empty.',
      exitCode: 1,
    })

    expect(mocks.prComment).not.toHaveBeenCalled()
  })

  it('pr comment rejects invalid pull request ids before posting', async () => {
    await expect(
      createProgram(prCommand).parseAsync([
        'node',
        'bbk',
        'pr',
        'comment',
        'abc',
        '--message',
        'Revisa este caso borde',
      ]),
    ).rejects.toMatchObject({
      message: 'Invalid pull request ID: abc',
      exitCode: 1,
    })

    expect(mocks.prComment).not.toHaveBeenCalled()
  })

  it('pr comment maps a missing pull request to a user-facing error', async () => {
    mocks.prComment.mockRejectedValue(new BitbucketApiError('Not Found', 404, '/pullrequests/404/comments'))

    await expect(
      createProgram(prCommand).parseAsync([
        'node',
        'bbk',
        'pr',
        'comment',
        '404',
        '--message',
        'Revisa este caso borde',
      ]),
    ).rejects.toMatchObject({
      message: 'PR #404 not found.',
      exitCode: 1,
    })
  })

  it('pr files renders human-readable changed files', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'files', '123'])

    expect(mocks.prFiles).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('Status'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('modified'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('src/old-name.ts'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('src/new-name.ts'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('12'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('3'))
    spy.mockRestore()
  })

  it('pr files outputs raw JSON when requested', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'files', '123', '--json'])

    expect(mocks.prFiles).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('"status": "modified"'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('"path": "src/new-name.ts"'))
    spy.mockRestore()
  })

  it('pr files reports when a pull request has no changed files', async () => {
    mocks.prFiles.mockResolvedValue([])
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'files', '123'])

    expect(mocks.prFiles).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(spy).toHaveBeenCalledWith('PR #123 has no changed files.')
    spy.mockRestore()
  })

  it('pr files rejects invalid pull request ids before fetching files', async () => {
    await expect(
      createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'files', 'abc']),
    ).rejects.toMatchObject({
      message: 'Invalid pull request ID: abc',
      exitCode: 1,
    })

    expect(mocks.prFiles).not.toHaveBeenCalled()
  })

  it('pr files maps a missing pull request to a user-facing error', async () => {
    mocks.prFiles.mockRejectedValue(new BitbucketApiError('Not Found', 404, '/pullrequests/404'))

    await expect(
      createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'files', '404']),
    ).rejects.toMatchObject({
      message: 'PR #404 not found.',
      exitCode: 1,
    })
  })

  it('pr diff outputs raw unified diff text', async () => {
    const diff = 'diff --git a/src/app.ts b/src/app.ts\n+console.log("new")\n'
    mocks.prDiff.mockResolvedValue(diff)
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'diff', '123'])

    expect(mocks.prDiff).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(spy).toHaveBeenCalledWith(diff)
    spy.mockRestore()
  })

  it('pr diff reports when a pull request has no diff', async () => {
    mocks.prDiff.mockResolvedValue('   \n')
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'diff', '123'])

    expect(mocks.prDiff).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(spy).toHaveBeenCalledWith('PR #123 has no diff changes.')
    spy.mockRestore()
  })

  it('pr diff rejects invalid pull request ids before fetching the diff', async () => {
    await expect(
      createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'diff', 'abc']),
    ).rejects.toMatchObject({
      message: 'Invalid pull request ID: abc',
      exitCode: 1,
    })

    expect(mocks.prDiff).not.toHaveBeenCalled()
  })

  it('pr diff maps a missing pull request to a user-facing error', async () => {
    mocks.prDiff.mockRejectedValue(new BitbucketApiError('Not Found', 404, '/pullrequests/404/diff'))

    await expect(
      createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'diff', '404']),
    ).rejects.toMatchObject({
      message: 'PR #404 not found.',
      exitCode: 1,
    })
  })

  it('pr merge merges the pull request without prompting when --yes is provided', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'merge', '123', '--yes'])

    expect(mocks.clackConfirm).not.toHaveBeenCalled()
    expect(mocks.prMerge).toHaveBeenCalledWith('testworkspace', 'test-repo', 123, {})
    expect(spy).toHaveBeenCalledWith(
      expect.any(String),
      'Merged PR #123: https://bitbucket.org/ws/repo/pull-requests/123',
    )
    spy.mockRestore()
  })

  it('pr merge passes verified strategy and message options to the service', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync([
      'node',
      'bbk',
      'pr',
      'merge',
      '123',
      '--yes',
      '--strategy',
      'squash',
      '--message',
      'Merge feature X',
    ])

    expect(mocks.prMerge).toHaveBeenCalledWith('testworkspace', 'test-repo', 123, {
      merge_strategy: 'squash',
      message: 'Merge feature X',
    })
    expect(spy).toHaveBeenCalledWith(
      expect.any(String),
      'Merged PR #123: https://bitbucket.org/ws/repo/pull-requests/123',
    )
    spy.mockRestore()
  })

  it('pr merge cancellation does not call the merge service', async () => {
    mocks.clackConfirm.mockResolvedValue(false)
    const spy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'merge', '123'])

    expect(mocks.clackConfirm).toHaveBeenCalledWith({
      message: 'Merge PR #123 into testworkspace/test-repo?',
      initialValue: false,
    })
    expect(mocks.prMerge).not.toHaveBeenCalled()
    expect(spy).toHaveBeenCalledWith(expect.any(String), 'Merge cancelled. PR #123 was not merged.')
    spy.mockRestore()
  })

  it('pr merge rejects invalid pull request ids before merging', async () => {
    await expect(
      createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'merge', 'abc', '--yes']),
    ).rejects.toMatchObject({
      message: 'Invalid pull request ID: abc',
      exitCode: 1,
    })

    expect(mocks.clackConfirm).not.toHaveBeenCalled()
    expect(mocks.prMerge).not.toHaveBeenCalled()
  })

  it('pr merge rejects unsupported merge strategies before merging', async () => {
    await expect(
      createProgram(prCommand).parseAsync([
        'node',
        'bbk',
        'pr',
        'merge',
        '123',
        '--yes',
        '--strategy',
        'octopus',
      ]),
    ).rejects.toMatchObject({
      message:
        'Invalid merge strategy: octopus. Supported strategies: merge_commit, squash, fast_forward, squash_fast_forward, rebase_fast_forward, rebase_merge.',
      exitCode: 1,
    })

    expect(mocks.prMerge).not.toHaveBeenCalled()
  })

  it('pr merge maps a missing pull request to a user-facing error', async () => {
    mocks.prMerge.mockRejectedValue(new BitbucketApiError('Not Found', 404, '/pullrequests/404'))

    await expect(
      createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'merge', '404', '--yes']),
    ).rejects.toMatchObject({
      message: 'PR #404 not found.',
      exitCode: 1,
    })
  })

  it('pr view renders human-readable pull request details', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'view', '123'])

    expect(mocks.prGet).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('Workflow PR'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('feature/checkout-pr → main'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('https://bitbucket.org/ws/repo/pull-requests/123'))
    spy.mockRestore()
  })

  it('pr view outputs raw JSON when requested', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'view', '123', '--json'])

    expect(mocks.prGet).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('"title": "Workflow PR"'))
    spy.mockRestore()
  })

  it('pr view rejects invalid ids and maps not found errors', async () => {
    await expect(createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'view', 'abc'])).rejects.toMatchObject({
      message: 'Invalid pull request ID: abc',
      exitCode: 1,
    })
    expect(mocks.prGet).not.toHaveBeenCalled()

    mocks.prGet.mockRejectedValue(new BitbucketApiError('Not Found', 404, '/pullrequests/404'))
    await expect(createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'view', '404'])).rejects.toMatchObject({
      message: 'PR #404 not found.',
      exitCode: 1,
    })
  })

  it('pr status renders PR metadata summary without treating checks as status', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'status', '123'])

    expect(mocks.prGet).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(mocks.prChecks).not.toHaveBeenCalled()
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('OPEN'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('Tasks'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('Approvals'))
    spy.mockRestore()
  })

  it('pr status supports JSON, rejects invalid ids, and maps missing PRs', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'status', '123', '--json'])

    expect(mocks.prGet).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('"task_count": 2'))
    spy.mockRestore()

    await expect(createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'status', 'abc'])).rejects.toMatchObject({
      message: 'Invalid pull request ID: abc',
      exitCode: 1,
    })

    mocks.prGet.mockRejectedValue(new BitbucketApiError('Not Found', 404, '/pullrequests/404'))
    await expect(createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'status', '404'])).rejects.toMatchObject({
      message: 'PR #404 not found.',
      exitCode: 1,
    })
  })

  it('pr commits renders commits and reports empty commit pages', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'commits', '123'])

    expect(mocks.prCommits).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('abcdef1'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('Add workflow commands'))

    mocks.prCommits.mockResolvedValue([])
    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'commits', '123'])
    expect(spy).toHaveBeenCalledWith('PR #123 has no commits.')
    spy.mockRestore()
  })

  it('pr reviewers renders reviewers and reports when none are present', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'reviewers', '123'])

    expect(mocks.prGet).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('Jane Reviewer'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('John Approver'))

    mocks.prGet.mockResolvedValue({
      id: 124,
      title: 'No reviewers',
      state: 'OPEN',
      source: { branch: { name: 'feature/no-reviewers' }, repository: { full_name: 'ws/repo' } },
      destination: { branch: { name: 'main' } },
      author: { display_name: 'Jane Doe' },
      created_on: '2026-04-28T00:00:00.000Z',
      updated_on: '2026-04-28T01:00:00.000Z',
      links: { html: { href: 'https://bitbucket.org/ws/repo/pull-requests/124' } },
    })
    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'reviewers', '124'])
    expect(spy).toHaveBeenCalledWith('PR #124 has no reviewers.')
    spy.mockRestore()
  })

  it('pr tasks renders pull request tasks and reports empty task pages', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'tasks', '123'])

    expect(mocks.prTasks).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('501'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('Add tests'))

    mocks.prTasks.mockResolvedValue([])
    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'tasks', '123'])
    expect(spy).toHaveBeenCalledWith('PR #123 has no tasks.')
    spy.mockRestore()
  })

  it('pr task create trims the message, calls the service, and rejects blank messages', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync([
      'node',
      'bbk',
      'pr',
      'task',
      'create',
      '123',
      '--message',
      '  Review auth flow  ',
    ])

    expect(mocks.prCreateTask).toHaveBeenCalledWith('testworkspace', 'test-repo', 123, 'Review auth flow')
    expect(spy).toHaveBeenCalledWith(expect.any(String), 'Task #502 created on PR #123: Review auth flow')
    spy.mockRestore()

    await expect(
      createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'task', 'create', '123', '--message', '   ']),
    ).rejects.toMatchObject({
      message: 'Task message cannot be empty.',
      exitCode: 1,
    })
  })

  it('pr task resolve validates ids, calls the service, and displays confirmation', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'task', 'resolve', '123', '502'])

    expect(mocks.prResolveTask).toHaveBeenCalledWith('testworkspace', 'test-repo', 123, 502)
    expect(spy).toHaveBeenCalledWith(expect.any(String), 'Task #502 resolved on PR #123.')
    spy.mockRestore()

    await expect(createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'task', 'resolve', 'abc', '502'])).rejects.toMatchObject({
      message: 'Invalid pull request ID: abc',
      exitCode: 1,
    })
    await expect(createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'task', 'resolve', '123', 'nope'])).rejects.toMatchObject({
      message: 'Invalid task ID: nope',
      exitCode: 1,
    })
  })

  it('pr checks renders commit statuses and reports empty status pages', async () => {
    const spy = vi.spyOn(console, 'log').mockImplementation(() => {})

    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'checks', '123'])

    expect(mocks.prChecks).toHaveBeenCalledWith('testworkspace', 'test-repo', 123)
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('SUCCESSFUL'))
    expect(spy).toHaveBeenCalledWith(expect.stringContaining('Build passed'))

    mocks.prChecks.mockResolvedValue([])
    await createProgram(prCommand).parseAsync(['node', 'bbk', 'pr', 'checks', '123'])
    expect(spy).toHaveBeenCalledWith('PR #123 has no checks.')
    spy.mockRestore()
  })

  it('branch list maps unexpected failures to exit code 2 cli errors', async () => {
    mocks.branchList.mockRejectedValue(new Error('boom'))

    await expect(
      createProgram(branchCommand).parseAsync(['node', 'bbk', 'branch', 'list']),
    ).rejects.toMatchObject({
      message: 'An unexpected error occurred.',
      exitCode: 2,
    })
  })
})
