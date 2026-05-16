import type { BitbucketClient } from '../client/bitbucket.client.js'
import type { BitbucketBranch } from '../types/bitbucket.types.js'

const DAY_IN_MS = 24 * 60 * 60 * 1000

export class BranchService {
  constructor(
    private readonly client: Pick<BitbucketClient, 'paginate'>,
  ) {}

  async list(workspace: string, repoSlug: string): Promise<BitbucketBranch[]> {
    const branches: BitbucketBranch[] = []

    for await (const branch of this.client.paginate<BitbucketBranch>(
      `/repositories/${workspace}/${repoSlug}/refs/branches`,
    )) {
      branches.push(branch)
    }

    return branches
  }

  async stale(workspace: string, repoSlug: string, days: number): Promise<BitbucketBranch[]> {
    const branches = await this.list(workspace, repoSlug)
    const cutoff = Date.now() - days * DAY_IN_MS

    return branches.filter((branch) => new Date(branch.target.date).getTime() < cutoff)
  }
}
