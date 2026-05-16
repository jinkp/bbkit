import { execFileSync } from 'node:child_process'
import { CliError } from '../errors/cli.errors.js'

export interface GitCheckoutResult {
  branchName: string
}

export class GitCheckoutService {
  checkoutSourceBranch(branchName: string): GitCheckoutResult {
    if (!isSafeBranchName(branchName)) {
      throw new CliError(`Unsafe source branch name: '${branchName}'.`, 1)
    }

    try {
      execFileSync('git', ['fetch', 'origin', `${branchName}:refs/heads/${branchName}`], {
        stdio: 'inherit',
      })
      execFileSync('git', ['checkout', branchName], { stdio: 'inherit' })
    } catch {
      throw new CliError(`Could not check out source branch '${branchName}'.`, 1)
    }

    return { branchName }
  }
}

function isSafeBranchName(branchName: string): boolean {
  const trimmed = branchName.trim()
  return (
    trimmed.length > 0 &&
    trimmed === branchName &&
    !trimmed.startsWith('-') &&
    !hasUnsafeBranchNameCharacter(trimmed)
  )
}

function hasUnsafeBranchNameCharacter(branchName: string): boolean {
  return [...branchName].some((character) => {
    const codePoint = character.codePointAt(0) ?? 0
    return codePoint <= 0x1f || codePoint === 0x7f || /\s/u.test(character)
  })
}
