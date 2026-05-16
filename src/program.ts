import { Command } from 'commander'
import { authCommand } from './commands/auth.command.js'
import { branchCommand } from './commands/branch.command.js'
import { configCommand } from './commands/config.command.js'
import { pipelineCommand } from './commands/pipeline.command.js'
import { prCommand } from './commands/pr.command.js'
import { repoCommand } from './commands/repo.command.js'
import { setupCommand } from './commands/setup.command.js'
import { createVersionCommand } from './commands/version.command.js'
import { packageMetadata, type PackageMetadata } from './version.js'

export function createCliProgram(metadata: PackageMetadata = packageMetadata): Command {
  const program = new Command()

  program
    .name('bbk')
    .description('A modern CLI for Bitbucket Cloud')
    .version(metadata.version)

  program.addCommand(createVersionCommand(metadata))
  program.addCommand(setupCommand)
  program.addCommand(authCommand)
  program.addCommand(configCommand)
  program.addCommand(repoCommand)
  program.addCommand(prCommand)
  program.addCommand(branchCommand)
  program.addCommand(pipelineCommand)

  return program
}
