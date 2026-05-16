import { execFileSync } from 'node:child_process'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { CliError } from '../src/errors/cli.errors.js'
import { GitCheckoutService } from '../src/services/git-checkout.service.js'

vi.mock('node:child_process', () => ({
  execFileSync: vi.fn(),
}))

describe('GitCheckoutService', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('fetches the remote PR source branch and checks it out locally', () => {
    const service = new GitCheckoutService()

    const result = service.checkoutSourceBranch('feature/checkout-pr')

    expect(execFileSync).toHaveBeenNthCalledWith(
      1,
      'git',
      ['fetch', 'origin', 'feature/checkout-pr:refs/heads/feature/checkout-pr'],
      { stdio: 'inherit' },
    )
    expect(execFileSync).toHaveBeenNthCalledWith(
      2,
      'git',
      ['checkout', 'feature/checkout-pr'],
      { stdio: 'inherit' },
    )
    expect(result).toEqual({ branchName: 'feature/checkout-pr' })
  })

  it('rejects unsafe branch names before invoking git', () => {
    const service = new GitCheckoutService()

    expect(() => service.checkoutSourceBranch('-bad-branch')).toThrow(CliError)
    expect(() => service.checkoutSourceBranch('feature/bad branch')).toThrow(CliError)
    expect(() => service.checkoutSourceBranch('feature/bad\nbranch')).toThrow(CliError)
    expect(execFileSync).not.toHaveBeenCalled()
  })

  it('maps git execution failures to a user-facing CliError', () => {
    vi.mocked(execFileSync).mockImplementation(() => {
      throw new Error('git failed')
    })
    const service = new GitCheckoutService()

    expect(() => service.checkoutSourceBranch('feature/missing')).toThrow(
      new CliError("Could not check out source branch 'feature/missing'.", 1),
    )
  })
})
