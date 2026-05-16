import { Command } from 'commander'
import * as p from '@clack/prompts'
import { authService } from '../services/auth.service.js'
import { CliError } from '../errors/cli.errors.js'
import { logger } from '../utils/logger.js'
import { getGitCredential, getGitUserEmail, isEmail } from '../utils/git-credential.js'
import { inferFromGitRemote } from '../utils/git-remote.js'
import { configStore } from '../config/config.store.js'

export const authCommand = new Command('auth').description('Manage Bitbucket authentication')

authCommand
  .command('login')
  .description('Authenticate with Bitbucket Cloud using an API token')
  .action(async () => {
    try {
      // Try to pre-fill email from git credential helper or git config
      const gitCred = getGitCredential('bitbucket.org')
      const gitEmail = getGitUserEmail()

      // Prefer email from credential helper if it looks like an email,
      // otherwise fall back to git config user.email
      const detectedEmail =
        (gitCred?.username && isEmail(gitCred.username) ? gitCred.username : null) ??
        (gitEmail && isEmail(gitEmail) ? gitEmail : null)

      if (detectedEmail) {
        logger.info(`Detected Atlassian account: ${detectedEmail}`)
      }

      // Try to detect workspace from git remote
      const detectedWorkspace =
        process.env['BITBUCKET_WORKSPACE'] ??
        configStore.getWorkspace() ??
        inferFromGitRemote()?.workspace

      const answers = await p.group(
        {
          username: () =>
            p.text({
              message: 'Atlassian email (e.g. you@company.com)',
              initialValue: detectedEmail ?? undefined,
              validate: (v) => {
                if (!v?.trim()) return 'Email is required'
                if (!v.includes('@')) return 'Enter a valid Atlassian email address'
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
          workspace: () =>
            p.text({
              message: 'Default Bitbucket workspace slug (e.g. my-company)',
              initialValue: detectedWorkspace ?? undefined,
              validate: (v) => {
                if (!v?.trim()) return 'Workspace is required'
              },
            }),
        },
        {
          onCancel: () => {
            p.cancel('Login cancelled.')
            process.exit(0)
          },
        },
      )

      const spin = p.spinner()
      spin.start('Verifying credentials...')

      try {
        await authService.login({ username: answers.username, apiToken: answers.apiToken })
        spin.stop('Credentials verified')
      } catch {
        spin.stop('Credentials failed')
        p.log.error('Invalid credentials. Check your email and API token.')
        p.log.info(
          'Generate a token at: https://id.atlassian.com/manage-profile/security/api-tokens',
        )
        process.exit(1)
      }

      configStore.setWorkspace(answers.workspace)
      logger.success(`Logged in as ${answers.username}`)
      logger.success(`Default workspace set to: ${answers.workspace}`)
      logger.info('Tip: workspace slug is the identifier in https://bitbucket.org/{workspace}/')
    } catch (err) {
      if (err instanceof CliError) {
        logger.error(err.message)
        process.exit(err.exitCode)
      }

      logger.error('Login failed unexpectedly.')
      process.exit(2)
    }
  })

authCommand
  .command('status')
  .description('Show current authentication status')
  .action(async () => {
    const status = await authService.status()
    if (status.authenticated) {
      logger.success(`Authenticated as ${status.username} (via ${status.source})`)
    } else {
      logger.warn('Not authenticated. Run `bbk auth login`.')
    }
  })

authCommand
  .command('logout')
  .description('Remove stored credentials')
  .action(async () => {
    await authService.logout()
    logger.success('Logged out successfully.')
  })
