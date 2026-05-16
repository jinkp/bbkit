import { describe, expect, it, vi } from 'vitest'
import { BranchService } from '../src/services/branch.service.js'
import type { BitbucketBranch } from '../src/types/bitbucket.types.js'

async function* createBranchPages(items: BitbucketBranch[]) {
  for (const item of items) {
    yield item
  }
}

describe('BranchService', () => {
  it('lists repository branches with pagination', async () => {
    const branches: BitbucketBranch[] = [
      {
        name: 'main',
        target: {
          date: '2026-04-28T00:00:00.000Z',
          hash: 'abcdef1234567890',
          author: { user: { display_name: 'Jane Doe' } },
        },
      },
    ]

    const client = {
      paginate: vi.fn(() => createBranchPages(branches)),
    }

    const service = new BranchService(client)
    const result = await service.list('workspace', 'repo')

    expect(client.paginate).toHaveBeenCalledWith('/repositories/workspace/repo/refs/branches')
    expect(result).toEqual(branches)
  })

  it('filters stale branches older than the requested days', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-28T00:00:00.000Z'))

    const branches: BitbucketBranch[] = [
      {
        name: 'stale-branch',
        target: {
          date: '2026-01-01T00:00:00.000Z',
          hash: 'abcdef1234567890',
          author: { user: { display_name: 'Jane Doe' } },
        },
      },
      {
        name: 'fresh-branch',
        target: {
          date: '2026-04-15T00:00:00.000Z',
          hash: '1234567890abcdef',
          author: { user: { display_name: 'John Doe' } },
        },
      },
    ]

    const client = {
      paginate: vi.fn(() => createBranchPages(branches)),
    }

    const service = new BranchService(client)
    const result = await service.stale('workspace', 'repo', 60)

    expect(result).toEqual([branches[0]])

    vi.useRealTimers()
  })
})
