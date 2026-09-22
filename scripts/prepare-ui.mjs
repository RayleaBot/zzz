import fs from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

// Copies the RayleaBot Vue SDK the management page builds against into
// .rayleabot/sdk/vue; the SDK directory defaults to the adjacent checkout.
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const sdk = path.resolve(process.argv[2] || path.join(root, '../../RayleaBot/sdk/vue'))
const manifest = JSON.parse(await fs.readFile(path.join(sdk, 'package.json'), 'utf8'))
if (manifest.name !== '@rayleabot/plugin-ui') throw new Error('Select the RayleaBot Vue SDK directory.')
const destination = path.join(root, '.rayleabot/sdk/vue')
await fs.mkdir(destination, { recursive: true })
for (const name of ['src', 'package.json', 'tsconfig.json', 'tsconfig.build.json']) {
  await fs.cp(path.join(sdk, name), path.join(destination, name), { recursive: true })
}
process.stdout.write('Prepared the Vue SDK.\n')
