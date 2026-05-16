import { BitbucketApiError, mapNetworkErrorToCliError } from '../errors/cli.errors.js'
import type { PaginatedResponse } from '../types/bitbucket.types.js'

const BASE_URL = 'https://api.bitbucket.org/2.0'

export class BitbucketClient {
  private authHeader: string

  constructor(username: string, apiToken: string) {
    this.authHeader = `Basic ${Buffer.from(`${username}:${apiToken}`).toString('base64')}`
  }

  private async request<T>(url: string, options: RequestInit = {}): Promise<T> {
    const headers = {
      'Authorization': this.authHeader,
      'Content-Type': 'application/json',
      'Accept': 'application/json',
      ...options.headers,
    }

    let response: Response
    try {
      response = await fetch(url, { ...options, headers })
    } catch {
      throw mapNetworkErrorToCliError()
    }

    if (!response.ok) {
      const body = await response.text().catch(() => '')
      throw new BitbucketApiError(body || response.statusText, response.status, url)
    }

    return response.json() as Promise<T>
  }

  private async requestText(url: string, options: RequestInit = {}): Promise<string> {
    const headers = {
      'Authorization': this.authHeader,
      'Content-Type': 'application/json',
      'Accept': 'text/plain',
      ...options.headers,
    }

    let response: Response
    try {
      response = await fetch(url, { ...options, headers })
    } catch {
      throw mapNetworkErrorToCliError()
    }

    if (!response.ok) {
      const body = await response.text().catch(() => '')
      throw new BitbucketApiError(body || response.statusText, response.status, url)
    }

    return response.text()
  }

  async get<T>(path: string): Promise<T> {
    const url = path.startsWith('https://') ? path : `${BASE_URL}${path}`
    return this.request<T>(url)
  }

  async getText(path: string): Promise<string> {
    const url = path.startsWith('https://') ? path : `${BASE_URL}${path}`
    return this.requestText(url)
  }

  async post<T>(path: string, body: unknown): Promise<T> {
    const url = path.startsWith('https://') ? path : `${BASE_URL}${path}`
    return this.request<T>(url, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  }

  async patch<T>(path: string, body: unknown): Promise<T> {
    const url = path.startsWith('https://') ? path : `${BASE_URL}${path}`
    return this.request<T>(url, {
      method: 'PUT',
      body: JSON.stringify(body),
    })
  }

  async *paginate<T>(path: string): AsyncGenerator<T> {
    let url: string | undefined = path.startsWith('https://')
      ? path
      : `${BASE_URL}${path}`

    while (url) {
      const page: PaginatedResponse<T> = await this.get<PaginatedResponse<T>>(url)
      for (const item of page.values) {
        yield item
      }
      url = page.next
    }
  }
}
