#!/usr/bin/env node
// Builds the simulator to frontend/public/raft.wasm and copies the matching
// wasm_exec.js glue from the same Go toolchain. Plain Node, so it runs the
// same on Windows, macOS, Linux and CI.
import { execFileSync } from 'node:child_process'
import { copyFileSync, existsSync, mkdirSync, statSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const outDir = join(root, 'frontend', 'public')
const wasm = join(outDir, 'raft.wasm')

mkdirSync(outDir, { recursive: true })
execFileSync('go', ['build', '-trimpath', '-ldflags=-s -w', '-o', wasm, './cmd/wasm'], {
  cwd: join(root, 'backend'),
  env: { ...process.env, GOOS: 'js', GOARCH: 'wasm' },
  stdio: 'inherit',
})

// Go 1.24 moved wasm_exec.js from misc/wasm to lib/wasm.
const goroot = execFileSync('go', ['env', 'GOROOT']).toString().trim()
const glue = [join(goroot, 'lib', 'wasm', 'wasm_exec.js'), join(goroot, 'misc', 'wasm', 'wasm_exec.js')].find(existsSync)
if (!glue) throw new Error(`wasm_exec.js not found under ${goroot}`)
copyFileSync(glue, join(outDir, 'wasm_exec.js'))

console.log(`built ${wasm} (${(statSync(wasm).size / 1024 / 1024).toFixed(1)} MB)`)
