import type { BitbucketClient } from '../client/bitbucket.client.js'
import type { BitbucketPipeline, PaginatedResponse } from '../types/bitbucket.types.js'

export class PipelineService {
  constructor(
    private readonly client: Pick<BitbucketClient, 'get' | 'post'>,
  ) {}

  async list(
    workspace: string,
    repoSlug: string,
    branch?: string,
  ): Promise<BitbucketPipeline[]> {
    const query = branch
      ? `?sort=-created_on&pagelen=20&target.ref_name=${encodeURIComponent(branch)}`
      : '?sort=-created_on&pagelen=20'

    const response = await this.client.get<PaginatedResponse<BitbucketPipeline>>(
      `/repositories/${workspace}/${repoSlug}/pipelines/${query}`,
    )

    return response.values
  }

  run(
    workspace: string,
    repoSlug: string,
    branch: string,
  ): Promise<BitbucketPipeline> {
    return this.client.post<BitbucketPipeline>(
      `/repositories/${workspace}/${repoSlug}/pipelines/`,
      {
        target: {
          ref_type: 'branch',
          type: 'pipeline_ref_target',
          ref_name: branch,
        },
      },
    )
  }
}
