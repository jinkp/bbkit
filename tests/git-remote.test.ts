import { describe, it, expect, vi } from 'vitest'
import { execSync } from 'child_process'

vi.mock('child_process', () => ({
  execSync: vi.fn(),
}))

import { inferFromGitRemote } from '../src/utils/git-remote.js'

describe('inferFromGitRemote', () => {
  it('parses https bitbucket remote', () => {
    vi.mocked(execSync).mockReturnValue(
      'https://bitbucket.org/myworkspace/my-repo.git\n' as never,
    )

    const result = inferFromGitRemote()

    expect(result).toEqual({ workspace: 'myworkspace', repoSlug: 'my-repo' })
  })

  it('parses ssh bitbucket remote', () => {
    vi.mocked(execSync).mockReturnValue(
      'git@bitbucket.org:myworkspace/my-repo.git\n' as never,
    )

    const result = inferFromGitRemote()

    expect(result).toEqual({ workspace: 'myworkspace', repoSlug: 'my-repo' })
  })

  it('returns null for non-bitbucket remote', () => {
    vi.mocked(execSync).mockReturnValue('https://github.com/user/repo.git\n' as never)

    const result = inferFromGitRemote()

    expect(result).toBeNull()
  })

  it('returns null when git command fails', () => {
    vi.mocked(execSync).mockImplementation(() => {
      throw new Error('not a git repo')
    })

    const result = inferFromGitRemote()

    expect(result).toBeNull()
  })
})
