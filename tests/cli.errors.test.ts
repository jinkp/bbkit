import { describe, expect, it } from 'vitest'
import {
  BitbucketApiError,
  CliError,
  mapApiErrorToCliError,
  mapNetworkErrorToCliError,
} from '../src/errors/cli.errors.js'

describe('CliError', () => {
  it('defaults to exit code 1', () => {
    const err = new CliError('Something failed')

    expect(err.exitCode).toBe(1)
    expect(err.message).toBe('Something failed')
  })

  it('accepts custom exit code 2', () => {
    const err = new CliError('Unexpected', 2)

    expect(err.exitCode).toBe(2)
  })
})

describe('mapApiErrorToCliError', () => {
  it('maps 401 to auth message', () => {
    const err = mapApiErrorToCliError(new BitbucketApiError('Unauthorized', 401))

    expect(err.message).toContain('Authentication failed')
    expect(err.exitCode).toBe(1)
  })

  it('maps 404 to not found message', () => {
    const err = mapApiErrorToCliError(new BitbucketApiError('Not Found', 404, 'repository'))

    expect(err.message).toContain('not found')
    expect(err.message).toContain('repository')
  })

  it('maps network error', () => {
    const err = mapNetworkErrorToCliError()

    expect(err.message).toContain('Unable to reach')
    expect(err.exitCode).toBe(2)
  })
})
