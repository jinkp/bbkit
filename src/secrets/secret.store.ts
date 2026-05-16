import { createCipheriv, createDecipheriv, randomBytes, scryptSync } from 'node:crypto'
import { existsSync, mkdirSync, readFileSync, unlinkSync, writeFileSync } from 'node:fs'
import { homedir, hostname, userInfo } from 'node:os'
import { join } from 'node:path'
import { getEnvConfig, hasEnvCredentials } from '../config/env.js'

const SERVICE_NAME = 'bbkit-cli'
const ACCOUNT_NAME = 'api-token'
const SALT = 'bbkit-cli-v1-salt-2024'
const CREDENTIALS_DIR = join(homedir(), '.bbkit-cli')
const CREDENTIALS_FILE = join(CREDENTIALS_DIR, 'credentials.enc')

export interface Credentials {
  username: string
  apiToken: string
}

function deriveKey(): Buffer {
  const machineId = `${hostname()}-${userInfo().username}`
  return scryptSync(machineId, SALT, 32)
}

function encrypt(plaintext: string): string {
  const key = deriveKey()
  const iv = randomBytes(16)
  const cipher = createCipheriv('aes-256-gcm', key, iv)
  const encrypted = Buffer.concat([cipher.update(plaintext, 'utf8'), cipher.final()])
  const authTag = cipher.getAuthTag()

  return iv.toString('hex') + authTag.toString('hex') + encrypted.toString('hex')
}

function decrypt(ciphertext: string): string {
  const key = deriveKey()
  const iv = Buffer.from(ciphertext.slice(0, 32), 'hex')
  const authTag = Buffer.from(ciphertext.slice(32, 64), 'hex')
  const encrypted = Buffer.from(ciphertext.slice(64), 'hex')
  const decipher = createDecipheriv('aes-256-gcm', key, iv)

  decipher.setAuthTag(authTag)

  const decrypted = Buffer.concat([decipher.update(encrypted), decipher.final()])
  return decrypted.toString('utf8')
}

async function getKeytar() {
  // Skip keytar entirely when env vars are present — avoids native binding
  // crashes on Windows (UV_HANDLE_CLOSING) when keytar is imported unnecessarily
  if (hasEnvCredentials()) return null

  try {
    const keytar = await import('keytar')
    return keytar.default ?? keytar
  } catch {
    return null
  }
}

function saveToFile(credentials: Credentials): void {
  mkdirSync(CREDENTIALS_DIR, { recursive: true })
  writeFileSync(CREDENTIALS_FILE, encrypt(JSON.stringify(credentials)), 'utf8')
}

function loadFromFile(): Credentials | null {
  if (!existsSync(CREDENTIALS_FILE)) {
    return null
  }

  try {
    const payload = decrypt(readFileSync(CREDENTIALS_FILE, 'utf8'))
    return JSON.parse(payload) as Credentials
  } catch {
    return null
  }
}

function clearFile(): void {
  if (existsSync(CREDENTIALS_FILE)) {
    unlinkSync(CREDENTIALS_FILE)
  }
}

export const secretStore = {
  async save(credentials: Credentials): Promise<void> {
    const keytar = await getKeytar()

    if (keytar) {
      await keytar.setPassword(SERVICE_NAME, ACCOUNT_NAME, credentials.apiToken)
      return
    }

    saveToFile(credentials)
  },

  async get(): Promise<Credentials | null> {
    // Env vars = highest priority, no keytar or file needed
    if (hasEnvCredentials()) {
      const env = getEnvConfig()
      return { username: env.username!, apiToken: env.apiToken! }
    }

    // Try OS keychain (getKeytar already skips if env vars are set)
    try {
      const keytar = await getKeytar()
      if (keytar) {
        const token = await keytar.getPassword(SERVICE_NAME, ACCOUNT_NAME)
        if (token) {
          const { configStore } = await import('../config/config.store.js')
          const username = configStore.getUsername()
          if (username) return { username, apiToken: token }
        }
      }
    } catch {
      // keytar failed silently — fall through to file
    }

    // Encrypted file fallback
    return loadFromFile()
  },

  async clear(): Promise<void> {
    const keytar = await getKeytar()
    if (keytar) {
      await keytar.deletePassword(SERVICE_NAME, ACCOUNT_NAME)
    }

    clearFile()
  },

  async isAuthenticated(): Promise<boolean> {
    const creds = await this.get()
    return creds !== null
  },
}
