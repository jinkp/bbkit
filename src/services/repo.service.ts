import type { BitbucketClient } from '../client/bitbucket.client.js'
import type { BitbucketRepository } from '../types/bitbucket.types.js'

export class RepoService {
  constructor(
    private readonly client: Pick<BitbucketClient, 'paginate'>,
  ) {}

  async list(workspace: string): Promise<BitbucketRepository[]> {
    const repositories: BitbucketRepository[] = []

    for await (const repository of this.client.paginate<BitbucketRepository>(`/repositories/${workspace}`)) {
      repositories.push(repository)
    }

    return repositories
  }
}
