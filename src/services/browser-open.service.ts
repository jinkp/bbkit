import { execFileSync } from 'node:child_process'
import { CliError } from '../errors/cli.errors.js'

export interface BrowserOpenResult {
  url: string
}

export type BrowserOpenRunner = (
  file: string,
  args: readonly string[],
  options: { stdio: 'ignore' },
) => void

export class BrowserOpenService {
  constructor(
    private readonly runner: BrowserOpenRunner = execFileSync,
    private readonly platform: NodeJS.Platform = process.platform,
  ) {}

  openUrl(url: string): BrowserOpenResult {
    const safeUrl = resolveSafeHttpUrl(url)

    const [file, args] = this.resolveOpenCommand(safeUrl)

    try {
      this.runner(file, args, { stdio: 'ignore' })
    } catch {
      throw new CliError('Could not open URL in the default browser.', 1)
    }

    return { url: safeUrl }
  }

  private resolveOpenCommand(url: string): [file: string, args: string[]] {
    switch (this.platform) {
      case 'win32':
        return ['cmd', ['/c', 'start', '', url]]
      case 'darwin':
        return ['open', [url]]
      default:
        return ['xdg-open', [url]]
    }
  }
}

function resolveSafeHttpUrl(value: string): string {
  const trimmed = value.trim()

  if (trimmed.length === 0 || trimmed !== value) {
    throw new CliError('Pull request URL is invalid or unsafe.', 1)
  }

  let parsed: URL
  try {
    parsed = new URL(trimmed)
  } catch {
    throw new CliError('Pull request URL is invalid or unsafe.', 1)
  }

  if (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') {
    throw new CliError('Pull request URL is invalid or unsafe.', 1)
  }

  return parsed.toString()
}
