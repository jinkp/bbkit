import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('node:fs', () => ({
  mkdirSync: vi.fn(),
  readFileSync: vi.fn(),
  writeFileSync: vi.fn(),
  unlinkSync: vi.fn(),
  existsSync: vi.fn(),
}))

vi.mock('keytar', async () => {
  throw new Error('keytar not available')
})

import { existsSync, mkdirSync, readFileSync, unlinkSync, writeFileSync } from 'node:fs'
import { secretStore } from '../src/secrets/secret.store.js'

describe('secretStore file fallback', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    delete process.env['BITBUCKET_USERNAME']
    delete process.env['BITBUCKET_API_TOKEN']
  })

  afterEach(() => {
    delete process.env['BITBUCKET_USERNAME']
    delete process.env['BITBUCKET_API_TOKEN']
  })

  it('saves credentials to encrypted file when keytar unavailable', async () => {
    await secretStore.save({ username: 'testuser', apiToken: 'secret123' })

    expect(mkdirSync).toHaveBeenCalled()
    expect(writeFileSync).toHaveBeenCalled()

    const writtenContent = vi.mocked(writeFileSync).mock.calls[0]?.[1] as string
    expect(writtenContent).not.toContain('secret123')
    expect(writtenContent).not.toContain('testuser')
  })

  it('returns null when no credentials file exists', async () => {
    vi.mocked(existsSync).mockReturnValue(false)

    const result = await secretStore.get()

    expect(result).toBeNull()
  })

  it('returns credentials from encrypted file when keytar unavailable', async () => {
    let storedContent = ''

    vi.mocked(writeFileSync).mockImplementation((_path, content) => {
      storedContent = content as string
    })

    await secretStore.save({ username: 'testuser', apiToken: 'secret123' })

    vi.mocked(existsSync).mockReturnValue(true)
    vi.mocked(readFileSync).mockReturnValue(storedContent)

    const result = await secretStore.get()

    expect(result).toEqual({ username: 'testuser', apiToken: 'secret123' })
  })

  it('clears both keytar and file on logout', async () => {
    vi.mocked(existsSync).mockReturnValue(true)

    await secretStore.clear()

    expect(unlinkSync).toHaveBeenCalled()
  })

  it('returns null for corrupted credential file', async () => {
    vi.mocked(existsSync).mockReturnValue(true)
    vi.mocked(readFileSync).mockReturnValue('corrupted-not-valid-hex')

    const result = await secretStore.get()

    expect(result).toBeNull()
  })
})
