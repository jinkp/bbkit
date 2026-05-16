import { Command } from 'commander'
import { formatJson } from '../formatters/json.formatter.js'
import { createVersionDetails, type PackageMetadata } from '../version.js'

interface VersionCommandOptions {
  json?: boolean
}

export function createVersionCommand(metadata: PackageMetadata): Command {
  const command = new Command('version')

  command
    .description('Show the installed bbkit CLI version')
    .option('--json', 'Output version details as JSON')
    .action(function (this: Command, options: VersionCommandOptions) {
      const value = options.json
        ? formatJson(createVersionDetails(metadata))
        : metadata.version

      const output = command.parent?.configureOutput() ?? this.configureOutput()
      const writeOut = output.writeOut ?? ((text: string) => process.stdout.write(text))
      writeOut(`${value}\n`)
    })

  return command
}
