import { Command } from 'commander'
import { configStore } from '../config/config.store.js'
import { logger } from '../utils/logger.js'

const VALID_KEYS = ['workspace', 'username', 'defaultOutput'] as const
type ConfigKey = (typeof VALID_KEYS)[number]

function isValidKey(key: string): key is ConfigKey {
  return (VALID_KEYS as readonly string[]).includes(key)
}

export const configCommand = new Command('config').description(
  'View and manage bbkit configuration',
)

configCommand
  .command('list')
  .description('Show all configuration values')
  .action(() => {
    const entries = configStore.getAll()
    if (Object.keys(entries).length === 0) {
      logger.warn('No configuration set. Run `bbk setup` to get started.')
      return
    }
    for (const [key, value] of Object.entries(entries)) {
      logger.info(`${key} = ${value}`)
    }
  })

configCommand
  .command('get <key>')
  .description('Get a configuration value')
  .action((key: string) => {
    if (!isValidKey(key)) {
      logger.error(`Unknown config key: ${key}`)
      logger.info(`Valid keys: ${VALID_KEYS.join(', ')}`)
      process.exit(1)
    }

    const value = configStore.getValue(key)
    if (value === undefined) {
      logger.warn(`${key} is not set`)
      process.exit(1)
    }
    // Print raw value for scripting (no prefix)
    console.log(value)
  })

configCommand
  .command('set <key> <value>')
  .description('Set a configuration value')
  .action((key: string, value: string) => {
    if (!isValidKey(key)) {
      logger.error(`Unknown config key: ${key}`)
      logger.info(`Valid keys: ${VALID_KEYS.join(', ')}`)
      process.exit(1)
    }

    if (key === 'defaultOutput' && value !== 'table' && value !== 'json') {
      logger.error('defaultOutput must be "table" or "json"')
      process.exit(1)
    }

    configStore.setValue(key, value)
    logger.success(`${key} = ${value}`)
  })
