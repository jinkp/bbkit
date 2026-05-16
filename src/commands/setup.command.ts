import { Command } from 'commander'
import * as p from '@clack/prompts'
import { authService } from '../services/auth.service.js'
import { configStore } from '../config/config.store.js'
import { BitbucketClient } from '../client/bitbucket.client.js'
import { getGitCredential, getGitUserEmail, isEmail } from '../utils/git-credential.js'
import { inferFromGitRemote } from '../utils/git-remote.js'
import type { BitbucketWorkspace, PaginatedResponse } from '../types/bitbucket.types.js'

export const setupCommand = new Command('setup')
  .description('Interactive setup wizard for bbkit')
  .action(runSetupWizard)

async function runSetupWizard(): Promise<void> {
  p.intro('bbkit — Bitbucket Cloud CLI')

  // ── Detect defaults ──────────────────────────────────────────────────────
  const gitCred = getGitCredential('bitbucket.org')
  const gitEmail = getGitUserEmail()
  const detectedEmail =
    (gitCred?.username && isEmail(gitCred.username) ? gitCred.username : null) ??
    (gitEmail && isEmail(gitEmail) ? gitEmail : null)

  const detectedWorkspace =
    configStore.getWorkspace() ?? inferFromGitRemote()?.workspace

  // ── Step 1: Credentials ──────────────────────────────────────────────────
  const credentials = await p.group(
    {
      email: () =>
        p.text({
          message: 'Atlassian email',
          placeholder: 'you@company.com',
          initialValue: detectedEmail ?? undefined,
          validate: (v) => {
            if (!v?.trim()) return 'Email is required'
            if (!v.includes('@')) return 'Enter a valid email address'
          },
        }),
      apiToken: () =>
        p.password({
          message: 'Bitbucket API token',
          mask: '*',
          validate: (v) => {
            if (!v?.trim()) return 'API token is required'
          },
        }),
    },
    {
      onCancel: () => {
        p.cancel('Setup cancelled.')
        process.exit(0)
      },
    },
  )

  // ── Step 2: Validate credentials ─────────────────────────────────────────
  const spin = p.spinner()
  spin.start('Verifying credentials against Bitbucket API...')

  let client: BitbucketClient
  try {
    client = new BitbucketClient(credentials.email, credentials.apiToken)
    await client.get('/user')
    spin.stop('Credentials verified')
  } catch {
    spin.stop('Credentials failed')
    p.log.error('Invalid credentials. Check your email and API token.')
    p.log.info('Generate a token at: https://id.atlassian.com/manage-profile/security/api-tokens')
    p.outro('Setup failed')
    process.exit(1)
  }

  // Save credentials right away (validates + stores)
  await authService.login({ username: credentials.email, apiToken: credentials.apiToken })

  // ── Step 3: Workspace ────────────────────────────────────────────────────
  const workspace = await selectWorkspace(client, detectedWorkspace)

  // ── Step 4: Output format ────────────────────────────────────────────────
  const defaultOutput = await p.select({
    message: 'Default output format',
    options: [
      { value: 'table' as const, label: 'Table', hint: 'human-friendly (default)' },
      { value: 'json' as const, label: 'JSON', hint: 'machine-readable, piping' },
    ],
    initialValue: configStore.getDefaultOutput(),
  })

  if (p.isCancel(defaultOutput)) {
    p.cancel('Setup cancelled.')
    process.exit(0)
  }

  // ── Save config ──────────────────────────────────────────────────────────
  configStore.setWorkspace(workspace)
  configStore.setDefaultOutput(defaultOutput)

  // ── Summary ──────────────────────────────────────────────────────────────
  p.note(
    [
      `Email:      ${credentials.email}`,
      `Workspace:  ${workspace}`,
      `Output:     ${defaultOutput}`,
      `Credentials stored in OS keychain`,
    ].join('\n'),
    'Configuration saved',
  )

  p.outro('Ready! Run `bbk repo list` to get started.')
}

// ── Workspace selection ──────────────────────────────────────────────────────

async function selectWorkspace(
  client: BitbucketClient,
  detectedWorkspace: string | undefined,
): Promise<string> {
  // Try to fetch workspaces the user has access to
  const spin = p.spinner()
  spin.start('Fetching your workspaces...')

  let workspaces: BitbucketWorkspace[] = []
  try {
    const resp = await client.get<PaginatedResponse<BitbucketWorkspace>>(
      '/workspaces?pagelen=50',
    )
    workspaces = resp.values
    spin.stop(`Found ${workspaces.length} workspace(s)`)
  } catch {
    spin.stop('Could not fetch workspaces')
  }

  if (workspaces.length > 0) {
    // Build options from API, marking detected one
    const options = workspaces.map((ws) => ({
      value: ws.slug,
      label: ws.slug,
      hint: ws.name !== ws.slug ? ws.name : undefined,
    }))

    const selected = await p.select({
      message: 'Default workspace',
      options,
      initialValue: detectedWorkspace ?? workspaces[0]?.slug,
    })

    if (p.isCancel(selected)) {
      p.cancel('Setup cancelled.')
      process.exit(0)
    }

    return selected
  }

  // Fallback: manual input
  const manual = await p.text({
    message: 'Default workspace slug',
    placeholder: 'my-company',
    initialValue: detectedWorkspace ?? undefined,
    validate: (v) => {
      if (!v?.trim()) return 'Workspace is required'
    },
  })

  if (p.isCancel(manual)) {
    p.cancel('Setup cancelled.')
    process.exit(0)
  }

  return manual
}
