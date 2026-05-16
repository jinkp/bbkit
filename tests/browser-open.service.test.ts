import { describe, expect, it, vi } from 'vitest'
import { CliError } from '../src/errors/cli.errors.js'
import { BrowserOpenService, type BrowserOpenRunner } from '../src/services/browser-open.service.js'

describe('BrowserOpenService', () => {
  it('opens http URLs with the Windows default browser command without a shell string', () => {
    const runner: BrowserOpenRunner = vi.fn()
    const service = new BrowserOpenService(runner, 'win32')

    const result = service.openUrl('https://bitbucket.org/ws/repo/pull-requests/123')

    expect(runner).toHaveBeenCalledWith(
      'cmd',
      ['/c', 'start', '', 'https://bitbucket.org/ws/repo/pull-requests/123'],
      { stdio: 'ignore' },
    )
    expect(result).toEqual({ url: 'https://bitbucket.org/ws/repo/pull-requests/123' })
  })

  it('opens https URLs with the macOS default browser command', () => {
    const runner: BrowserOpenRunner = vi.fn()
    const service = new BrowserOpenService(runner, 'darwin')

    service.openUrl('https://bitbucket.org/ws/repo/pull-requests/456')

    expect(runner).toHaveBeenCalledWith(
      'open',
      ['https://bitbucket.org/ws/repo/pull-requests/456'],
      { stdio: 'ignore' },
    )
  })

  it('opens https URLs with the Linux default browser command', () => {
    const runner: BrowserOpenRunner = vi.fn()
    const service = new BrowserOpenService(runner, 'linux')

    service.openUrl('https://bitbucket.org/ws/repo/pull-requests/789')

    expect(runner).toHaveBeenCalledWith(
      'xdg-open',
      ['https://bitbucket.org/ws/repo/pull-requests/789'],
      { stdio: 'ignore' },
    )
  })

  it('rejects non-http URLs before invoking an opener command', () => {
    const runner: BrowserOpenRunner = vi.fn()
    const service = new BrowserOpenService(runner, 'linux')

    expect(() => service.openUrl('javascript:alert(1)')).toThrow(CliError)
    expect(runner).not.toHaveBeenCalled()
  })

  it('maps opener failures to a user-facing CliError', () => {
    const runner: BrowserOpenRunner = vi.fn(() => {
      throw new Error('opener failed')
    })
    const service = new BrowserOpenService(runner, 'linux')

    expect(() => service.openUrl('https://bitbucket.org/ws/repo/pull-requests/123')).toThrow(
      new CliError('Could not open URL in the default browser.', 1),
    )
  })
})
