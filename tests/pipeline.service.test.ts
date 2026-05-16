import { describe, expect, it, vi } from 'vitest'
import { PipelineService } from '../src/services/pipeline.service.js'
import type { BitbucketPipeline } from '../src/types/bitbucket.types.js'

describe('PipelineService', () => {
  it('lists recent pipelines and supports branch filtering', async () => {
    const pipelines: BitbucketPipeline[] = [
      {
        uuid: '{123}',
        build_number: 42,
        state: { name: 'COMPLETED', result: { name: 'SUCCESSFUL' } },
        target: { ref_name: 'main', ref_type: 'branch' },
        created_on: '2026-04-28T00:00:00.000Z',
        duration_in_seconds: 90,
        links: { self: { href: 'https://api.bitbucket.org/2.0/pipelines/123' } },
      },
    ]

    const client = {
      get: vi.fn().mockResolvedValue({ values: pipelines }),
      post: vi.fn(),
    }

    const service = new PipelineService(client)
    const result = await service.list('workspace', 'repo', 'feature/test')

    expect(client.get).toHaveBeenCalledWith(
      '/repositories/workspace/repo/pipelines/?sort=-created_on&pagelen=20&target.ref_name=feature%2Ftest',
    )
    expect(result).toEqual(pipelines)
  })

  it('starts a pipeline for the requested branch', async () => {
    const pipeline: BitbucketPipeline = {
      uuid: '{123}',
      build_number: 42,
      state: { name: 'PENDING' },
      target: { ref_name: 'develop', ref_type: 'branch' },
      created_on: '2026-04-28T00:00:00.000Z',
      links: { self: { href: 'https://api.bitbucket.org/2.0/pipelines/123' } },
    }

    const client = {
      get: vi.fn(),
      post: vi.fn().mockResolvedValue(pipeline),
    }

    const service = new PipelineService(client)
    const result = await service.run('workspace', 'repo', 'develop')

    expect(client.post).toHaveBeenCalledWith('/repositories/workspace/repo/pipelines/', {
      target: {
        ref_type: 'branch',
        type: 'pipeline_ref_target',
        ref_name: 'develop',
      },
    })
    expect(result).toEqual(pipeline)
  })
})
