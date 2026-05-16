import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../src/secrets/secret.store.js', () => ({
  secretStore: {
    save: vi.fn(),
    get: vi.fn(),
    clear: vi.fn(),
    isAuthenticated: vi.fn(),
  },
}))

vi.mock('../src/config/config.store.js', () => ({
  configStore: {
    getUsername: vi.fn(),
    setUsername: vi.fn(),
    getWorkspace: vi.fn(),
    setWorkspace: vi.fn(),
    getDefaultOutput: vi.fn().mockReturnValue('table'),
    clear: vi.fn(),
  },
}))

vi.mock('../src/client/bitbucket.client.js', () => ({
  BitbucketClient: vi.fn().mockImplementation(() => ({
    get: vi.fn().mockResolvedValue({ username: 'testuser', display_name: 'Test User' }),
  })),
}))

import { configStore } from '../src/config/config.store.js'
import { secretStore } from '../src/secrets/secret.store.js'
import { authService } from '../src/services/auth.service.js'

describe('authService', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    delete process.env['BITBUCKET_USERNAME']
    delete process.env['BITBUCKET_API_TOKEN']
  })

  describe('login', () => {
    it('saves credentials after successful API validation', async () => {
      await authService.login({ username: 'testuser', apiToken: 'token123' })

      expect(secretStore.save).toHaveBeenCalledWith({ username: 'testuser', apiToken: 'token123' })
      expect(configStore.setUsername).toHaveBeenCalledWith('testuser')
    })

    it('throws CliError when API validation fails', async () => {
      const { BitbucketClient } = await import('../src/client/bitbucket.client.js')
      vi.mocked(BitbucketClient).mockImplementationOnce(() => ({
        get: vi.fn().mockRejectedValue(new Error('401')),
        post: vi.fn(),
        paginate: vi.fn(),
      }))

      const { CliError } = await import('../src/errors/cli.errors.js')
      await expect(authService.login({ username: 'bad', apiToken: 'bad' })).rejects.toThrow(CliError)
    })
  })

  describe('status', () => {
    it('returns unauthenticated when no credentials exist', async () => {
      vi.mocked(secretStore.get).mockResolvedValue(null)

      const result = await authService.status()

      expect(result.authenticated).toBe(false)
    })

    it('returns authenticated with keychain source when credentials exist', async () => {
      vi.mocked(secretStore.get).mockResolvedValue({ username: 'testuser', apiToken: 'token123' })

      const result = await authService.status()

      expect(result.authenticated).toBe(true)
      expect(result.username).toBe('testuser')
      expect(result.source).toBe('keychain')
    })

    it('returns env source when env vars are set', async () => {
      process.env['BITBUCKET_USERNAME'] = 'envuser'
      process.env['BITBUCKET_API_TOKEN'] = 'envtoken'

      const result = await authService.status()

      expect(result.authenticated).toBe(true)
      expect(result.source).toBe('env')
    })
  })

  describe('logout', () => {
    it('clears stored credentials', async () => {
      await authService.logout()

      expect(secretStore.clear).toHaveBeenCalled()
    })
  })

  describe('getCredentialsOrFail', () => {
    it('throws CliError when not authenticated', async () => {
      vi.mocked(secretStore.get).mockResolvedValue(null)

      const { CliError } = await import('../src/errors/cli.errors.js')
      await expect(authService.getCredentialsOrFail()).rejects.toThrow(CliError)
    })

    it('returns credentials when authenticated', async () => {
      vi.mocked(secretStore.get).mockResolvedValue({ username: 'testuser', apiToken: 'token123' })

      const result = await authService.getCredentialsOrFail()

      expect(result.username).toBe('testuser')
    })
  })
})
