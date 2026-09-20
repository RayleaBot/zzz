import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
const script = fileURLToPath(new URL('../../game-plugin-kit/scripts/prepare-plugin.mjs', import.meta.url))
const root = fileURLToPath(new URL('..', import.meta.url))
const result = spawnSync(process.execPath, [script, root, ...process.argv.slice(2)], { stdio: 'inherit' })
process.exit(result.status ?? 1)
