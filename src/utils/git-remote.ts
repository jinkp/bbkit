import { execSync } from 'child_process'

export interface GitRemoteInfo {
  workspace: string
  repoSlug: string
}

export function inferFromGitRemote(): GitRemoteInfo | null {
  try {
    const remote = execSync('git remote get-url origin', { encoding: 'utf-8' }).trim()
    // Supports:
    // https://bitbucket.org/workspace/repo.git
    // git@bitbucket.org:workspace/repo.git
    const httpsMatch = remote.match(/bitbucket\.org\/([^/]+)\/([^/.]+)/)
    const sshMatch = remote.match(/bitbucket\.org:([^/]+)\/([^/.]+)/)
    const match = httpsMatch ?? sshMatch
    if (match) {
      return { workspace: match[1]!, repoSlug: match[2]! }
    }
    return null
  } catch {
    return null
  }
}
