import packageJson from '../package.json'

export interface PackageMetadata {
  name: string
  version: string
}

export interface VersionDetails extends PackageMetadata {
  node: string
  platform: NodeJS.Platform
  arch: NodeJS.Architecture
}

export const packageMetadata: PackageMetadata = {
  name: packageJson.name,
  version: packageJson.version,
}

export function createVersionDetails(metadata: PackageMetadata): VersionDetails {
  return {
    name: metadata.name,
    version: metadata.version,
    node: process.version,
    platform: process.platform,
    arch: process.arch,
  }
}
