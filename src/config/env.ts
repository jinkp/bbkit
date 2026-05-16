export interface EnvConfig {
  username?: string
  apiToken?: string
  workspace?: string
}

export function getEnvConfig(): EnvConfig {
  return {
    username: process.env['BITBUCKET_USERNAME'],
    apiToken: process.env['BITBUCKET_API_TOKEN'],
    workspace: process.env['BITBUCKET_WORKSPACE'],
  }
}

export function hasEnvCredentials(): boolean {
  const env = getEnvConfig()
  return Boolean(env.username && env.apiToken)
}
