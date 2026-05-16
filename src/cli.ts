#!/usr/bin/env node
import { CliError } from './errors/cli.errors.js'
import { createCliProgram } from './program.js'

const program = createCliProgram()

program.parseAsync(process.argv).catch((err: unknown) => {
  if (err instanceof CliError) {
    console.error(err.message)
    process.exit(err.exitCode)
  }
  console.error('An unexpected error occurred.')
  if (process.env['DEBUG']) {
    console.error(err)
  }
  process.exit(2)
})
