import { configStore } from '../config/config.store.js'
import { CliError } from '../errors/cli.errors.js'
import { secretStore, type Credentials } from '../secrets/secret.store.js'

export const authService = {
  async login(credentials: Credentials): Promise<void> {
    const { BitbucketClient } = await import('../client/bitbucket.client.js')
    const client = new BitbucketClient(credentials.username, credentials.apiToken)

    try {
      await client.get('/user')
    } catch {
      throw new CliError('Invalid credentials. Check your username and API token.', 1)
    }

    await secretStore.save(credentials)
    configStore.setUsername(credentials.username)
  },

  async status(): Promise<{ authenticated: boolean; username?: string; source?: string }> {
    const { getEnvConfig, hasEnvCredentials } = await import('../config/env.js')

    if (hasEnvCredentials()) {
      const env = getEnvConfig()
      return { authenticated: true, username: env.username, source: 'env' }
    }

    const creds = await secretStore.get()
    if (creds) {
      return { authenticated: true, username: creds.username, source: 'keychain' }
    }

    return { authenticated: false }
  },

  async logout(): Promise<void> {
    await secretStore.clear()
  },

  async getCredentialsOrFail(): Promise<Credentials> {
    const creds = await secretStore.get()
    if (!creds) {
      throw new CliError('Not authenticated. Run `bbk auth login`.', 1)
    }

    return creds
  },
}
