export class CliError extends Error {
  constructor(
    message: string,
    public readonly exitCode: 1 | 2 = 1,
  ) {
    super(message)
    this.name = 'CliError'
  }
}

export class BitbucketApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly context?: string,
  ) {
    super(message)
    this.name = 'BitbucketApiError'
  }
}

export function mapApiErrorToCliError(err: BitbucketApiError): CliError {
  switch (err.status) {
    case 401:
      return new CliError('Authentication failed. Run `bbk auth login`.', 1)
    case 403:
      return new CliError('Permission denied. Check your API token scopes.', 1)
    case 404:
      return new CliError(
        `Resource not found${err.context ? `: ${err.context}` : ''}.`,
        1,
      )
    default:
      return new CliError(`Bitbucket API error (${err.status}): ${err.message}`, 2)
  }
}

export function mapNetworkErrorToCliError(): CliError {
  return new CliError(
    'Unable to reach Bitbucket API. Check your connection.',
    2,
  )
}
