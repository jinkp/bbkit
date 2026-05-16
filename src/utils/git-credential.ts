import { execSync, spawnSync } from 'child_process'

export interface GitCredentialInfo {
  username: string
  /** password is the OAuth/git token — NOT usable as Bitbucket API token */
  password?: string
}

/**
 * Ask git's credential helper for stored credentials for bitbucket.org.
 * Works with Git Credential Manager (Windows/macOS/Linux), osxkeychain, etc.
 *
 * NOTE: The password returned here is an OAuth token that Git uses to push/pull.
 * It is NOT a Bitbucket API token and cannot be used for the REST API.
 * We only use the username (email) to pre-fill the login prompt.
 */
export function getGitCredential(host = 'bitbucket.org'): GitCredentialInfo | null {
  try {
    const result = spawnSync(
      'git',
      ['credential', 'fill'],
      {
        input: `protocol=https\nhost=${host}\n\n`,
        encoding: 'utf-8',
        timeout: 5000,
      },
    )

    if (result.status !== 0 || !result.stdout) return null

    const lines = result.stdout.split('\n')
    const get = (key: string) =>
      lines.find((l) => l.startsWith(`${key}=`))?.slice(key.length + 1).trim()

    const username = get('username')
    const password = get('password')

    if (!username) return null
    return { username, password }
  } catch {
    return null
  }
}

/**
 * Check if the username looks like an Atlassian email.
 * Git Credential Manager sometimes stores a nickname instead of email.
 */
export function isEmail(value: string): boolean {
  return value.includes('@')
}

/**
 * Try to resolve the Atlassian email from git config as fallback.
 * `git config user.email` is often the Atlassian account email.
 */
export function getGitUserEmail(): string | null {
  try {
    const email = execSync('git config user.email', { encoding: 'utf-8' }).trim()
    return email || null
  } catch {
    return null
  }
}
