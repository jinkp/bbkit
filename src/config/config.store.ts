import Conf from 'conf'

interface ConfigSchema {
  workspace?: string
  defaultOutput?: 'table' | 'json'
  username?: string
}

const config = new Conf<ConfigSchema>({
  projectName: 'bbkit-cli',
  schema: {
    workspace: { type: 'string' },
    defaultOutput: { type: 'string', enum: ['table', 'json'] },
    username: { type: 'string' },
  },
})

export const configStore = {
  getWorkspace(): string | undefined {
    return process.env['BITBUCKET_WORKSPACE'] ?? config.get('workspace')
  },
  setWorkspace(workspace: string): void {
    config.set('workspace', workspace)
  },
  getUsername(): string | undefined {
    return process.env['BITBUCKET_USERNAME'] ?? config.get('username')
  },
  setUsername(username: string): void {
    config.set('username', username)
  },
  getDefaultOutput(): 'table' | 'json' {
    return config.get('defaultOutput') ?? 'table'
  },
  setDefaultOutput(format: 'table' | 'json'): void {
    config.set('defaultOutput', format)
  },
  getAll(): Record<string, string> {
    const entries: Record<string, string> = {}
    const workspace = config.get('workspace')
    const username = config.get('username')
    const defaultOutput = config.get('defaultOutput')
    if (workspace) entries['workspace'] = workspace
    if (username) entries['username'] = username
    if (defaultOutput) entries['defaultOutput'] = defaultOutput
    return entries
  },
  getValue(key: keyof ConfigSchema): string | undefined {
    return config.get(key) as string | undefined
  },
  setValue(key: keyof ConfigSchema, value: string): void {
    config.set(key, value)
  },
  clear(): void {
    config.clear()
  },
}
