import { describe, expect, it, vi } from 'vitest'
import { RepoService } from '../src/services/repo.service.js'
import type { BitbucketRepository } from '../src/types/bitbucket.types.js'

async function* createRepositoryPages(items: BitbucketRepository[]) {
  for (const item of items) {
    yield item
  }
}

describe('RepoService', () => {
  it('returns all repositories from paginated responses', async () => {
    const repositories: BitbucketRepository[] = [
      {
        slug: 'repo-one',
        name: 'Repo One',
        language: 'TypeScript',
        updated_on: '2026-04-28T00:00:00.000Z',
        full_name: 'workspace/repo-one',
        is_private: false,
      },
      {
        slug: 'repo-two',
        name: 'Repo Two',
        language: 'JavaScript',
        updated_on: '2026-04-27T00:00:00.000Z',
        full_name: 'workspace/repo-two',
        is_private: true,
      },
    ]

    const client = {
      paginate: vi.fn(() => createRepositoryPages(repositories)),
    }

    const service = new RepoService(client)
    const result = await service.list('workspace')

    expect(client.paginate).toHaveBeenCalledWith('/repositories/workspace')
    expect(result).toEqual(repositories)
  })
})
