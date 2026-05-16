import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { BitbucketClient } from '../src/client/bitbucket.client.js'
import { CliError, BitbucketApiError } from '../src/errors/cli.errors.js'

describe('BitbucketClient', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('throws network CliError when fetch rejects', async () => {
    vi.mocked(fetch).mockRejectedValue(new Error('ENOTFOUND'))

    const client = new BitbucketClient('user', 'token')

    await expect(client.get('/user')).rejects.toThrow(CliError)
  })

  it('throws BitbucketApiError on non-ok response', async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response('Unauthorized', { status: 401 }),
    )

    const client = new BitbucketClient('user', 'token')

    await expect(client.get('/user')).rejects.toMatchObject({
      name: 'BitbucketApiError',
      status: 401,
      context: 'https://api.bitbucket.org/2.0/user',
    })
  })

  it('returns parsed JSON on success', async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify({ username: 'testuser' }), { status: 200 }),
    )

    const client = new BitbucketClient('user', 'token')
    const result = await client.get<{ username: string }>('/user')

    expect(result.username).toBe('testuser')
  })

  it('returns plain text responses for non-JSON endpoints', async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response('diff --git a/README.md b/README.md\n+new line\n', { status: 200 }),
    )

    const client = new BitbucketClient('user', 'token')
    const result = await client.getText('/repositories/ws/repo/pullrequests/123/diff')

    expect(result).toBe('diff --git a/README.md b/README.md\n+new line\n')
    expect(fetch).toHaveBeenCalledWith(
      'https://api.bitbucket.org/2.0/repositories/ws/repo/pullrequests/123/diff',
      expect.objectContaining({
        headers: expect.objectContaining({
          Accept: 'text/plain',
          Authorization: expect.stringMatching(/^Basic /),
        }),
      }),
    )
  })

  it('paginate yields all items across pages', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            values: [{ name: 'repo-1' }, { name: 'repo-2' }],
            next: 'https://api.bitbucket.org/2.0/repositories/ws?page=2',
          }),
          { status: 200 },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            values: [{ name: 'repo-3' }],
          }),
          { status: 200 },
        ),
      )

    const client = new BitbucketClient('user', 'token')
    const results: { name: string }[] = []

    for await (const item of client.paginate<{ name: string }>('/repositories/ws')) {
      results.push(item)
    }

    expect(results).toHaveLength(3)
    expect(results[2]?.name).toBe('repo-3')
  })
})
