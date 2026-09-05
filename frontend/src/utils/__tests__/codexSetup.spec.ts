// @vitest-environment node
import { afterEach, describe, expect, it } from 'vitest'
import { mkdtempSync, mkdirSync, readFileSync, readdirSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { spawnSync } from 'node:child_process'
import { escapeTomlBasicString, generateMacCodexCommand, generateWindowsCodexCommand } from '../codexSetup'

const roots: string[] = []
const config = 'model_provider = "OpenAI"\nmodel = "gpt-5.5"\n'
const auth = JSON.stringify({ OPENAI_API_KEY: 'sk-test' }, null, 2)
const pwsh = process.env.CODEX_TEST_PWSH || (process.platform === 'win32' ? 'powershell.exe' : 'pwsh')
const hasPowerShell = spawnSync(pwsh, ['-NoProfile', '-Command', '$PSVersionTable.PSVersion.ToString()']).status === 0

function sandbox() {
  const root = mkdtempSync(join(tmpdir(), 'codex-setup-'))
  roots.push(root)
  const bin = join(root, 'bin')
  const dir = join(root, '.codex')
  mkdirSync(bin)
  const stub = (name: string, body: string) => writeFileSync(join(bin, name), body, { mode: 0o755 })
  stub('pgrep', '#!/bin/sh\n[ "$CODEX_TEST_RUNNING" = "1" ]\n')
  stub('open', '#!/bin/sh\nexit 0\n')
  stub('osascript', '#!/usr/bin/env node\nconst fs = require("fs"); const value = JSON.parse(fs.readFileSync(0, "utf8")); if (Object.keys(value).length !== 1 || typeof value.OPENAI_API_KEY !== "string" || !value.OPENAI_API_KEY) process.exit(1);\n')
  stub('mv', '#!/bin/sh\ncase "$*" in *sub2api-auth*) if [ "$CODEX_TEST_FAIL_MOVE" = "1" ]; then echo "simulated auth replacement failure" >&2; exit 1; fi ;; esac\nexec /bin/mv "$@"\n')
  stub('cp', '#!/bin/sh\nif [ "$CODEX_TEST_FAIL_BACKUP" = "1" ]; then echo "simulated backup failure" >&2; exit 1; fi\nexec /bin/cp "$@"\n')
  const run = (env: Record<string, string> = {}, configContent = config, authContent = auth) => {
    const command = generateMacCodexCommand(configContent, authContent)
    const result = spawnSync('/bin/bash', ['-c', `${command}\nprintf '\\nPARENT_ALIVE\\n'`], {
      env: { ...process.env, HOME: root, PATH: `${bin}:${process.env.PATH}`, ...env }, encoding: 'utf8'
    })
    expect(result.status).toBe(0)
    expect(result.stdout).toContain('PARENT_ALIVE')
    return `${result.stdout}\n${result.stderr}`
  }
  return { root, dir, run }
}

afterEach(() => roots.splice(0).forEach((root) => rmSync(root, { recursive: true, force: true })))

describe('Codex setup scripts', () => {
  it('stops before creating files when the configuration directory is missing', () => {
    const { root, run } = sandbox()
    expect(run()).toContain('未找到 Codex 配置目录')
    expect(readdirSync(root)).toEqual(['bin'])
  })

  it('stops without writing when Codex is running', () => {
    const { dir, run } = sandbox()
    mkdirSync(dir)
    expect(run({ CODEX_TEST_RUNNING: '1' })).toContain('仍在运行')
    expect(readdirSync(dir)).toEqual([])
  })

  it.each([[], ['config.toml'], ['auth.json'], ['config.toml', 'auth.json']].map((existing) => ({ existing })))('writes both files and keeps unique backups: $existing', ({ existing }) => {
    const { dir, run } = sandbox()
    mkdirSync(dir)
    for (const file of existing) writeFileSync(join(dir, file), `old ${file}`)
    expect(run()).toContain('配置完成')
    expect(readFileSync(join(dir, 'config.toml'), 'utf8')).toBe(config)
    expect(readFileSync(join(dir, 'auth.json'), 'utf8')).toBe(auth)
    const firstBackups = readdirSync(dir).filter((name) => name.includes('.bak-'))
    expect(firstBackups).toHaveLength(existing.length)
    expect(run()).toContain('配置完成')
    expect(readdirSync(dir).filter((name) => name.includes('.bak-'))).toHaveLength(existing.length + 2)
    expect(readdirSync(dir).some((name) => name.startsWith('.sub2api-'))).toBe(false)
  })

  it.each([[], ['config.toml'], ['auth.json'], ['config.toml', 'auth.json']].map((existing) => ({ existing })))('rolls back both files after the second replacement fails: $existing', ({ existing }) => {
    const { dir, run } = sandbox()
    mkdirSync(dir)
    for (const file of existing) writeFileSync(join(dir, file), `old ${file}`)
    const output = run({ CODEX_TEST_FAIL_MOVE: '1' })
    expect(output).toContain('simulated auth replacement failure')
    expect(output).toContain('已恢复')
    expect(output).not.toContain('配置完成')
    expect(readdirSync(dir).filter((name) => !name.includes('.bak-')).sort()).toEqual(existing.slice().sort())
    for (const file of existing) expect(readFileSync(join(dir, file), 'utf8')).toBe(`old ${file}`)
  })

  it('does not replace files when backup or JSON validation fails', () => {
    const { dir, run } = sandbox()
    mkdirSync(dir)
    writeFileSync(join(dir, 'config.toml'), 'old')
    expect(run({ CODEX_TEST_FAIL_BACKUP: '1' })).toContain('simulated backup failure')
    expect(readFileSync(join(dir, 'config.toml'), 'utf8')).toBe('old')
    expect(run({}, config, '{invalid')).toContain('auth.json')
    expect(readFileSync(join(dir, 'config.toml'), 'utf8')).toBe('old')
    expect(readdirSync(dir)).not.toContain('auth.json')
  })

  it('preserves quoted and multiline payloads without executing shell text', () => {
    const { dir, run } = sandbox()
    mkdirSync(dir)
    const hostile = '\'"$HOME`echo bad`$(echo bad)\\\nSUB2API_CODEX_SETUP\n中文'
    const content = `model = "${escapeTomlBasicString(hostile)}"\n`
    const secret = JSON.stringify({ OPENAI_API_KEY: hostile }, null, 2)
    expect(run({}, content, secret)).toContain('配置完成')
    expect(readFileSync(join(dir, 'config.toml'), 'utf8')).toBe(content)
    expect(readFileSync(join(dir, 'auth.json'), 'utf8')).toBe(secret)
    const parsed = spawnSync('python3', ['-c', 'import sys, toml; print(toml.loads(sys.stdin.read())["model"])'], {
      input: content, encoding: 'utf8'
    })
    expect(parsed.status, parsed.stderr).toBe(0)
    expect(parsed.stdout.trim()).toBe(hostile)
    expect(escapeTomlBasicString('\n\r\t\u0000\u007f')).toBe('\\n\\r\\t\\u0000\\u007f')
  })

  it('isolates PowerShell, fails before writes, uses UTF-8 without BOM and includes rollback', () => {
    const command = generateWindowsCodexCommand(config, auth)
    expect(command.startsWith('& {\n')).toBe(true)
    expect(command).not.toMatch(/\bexit\b|New-Item|CreateDirectory/)
    expect(command).toContain('-PathType Container')
    expect(command.indexOf('-PathType Container')).toBeLessThan(command.indexOf('WriteAllText'))
    expect(command).toContain('UTF8Encoding]::new($false)')
    expect(command).toContain('$_.Exception.Message')
    expect(command).toContain('$replaceStarted')
    expect(command).toContain('已恢复')
  })
})

describe.skipIf(!hasPowerShell)('PowerShell execution', () => {
  function run(root: string, options: { running?: boolean; failReplace?: boolean; config?: string; auth?: string } = {}) {
    let command = generateWindowsCodexCommand(options.config ?? config, options.auth ?? auth)
    if (options.failReplace) command = command.replace('Replace-CodexFile $tempAuthFile $authFile', "throw 'simulated auth replacement failure'")
    const result = spawnSync(pwsh, ['-NoProfile', '-NonInteractive', '-Command', `
      function Get-Process { ${options.running ? "'Codex'" : ''} }
      function Invoke-Item {}
      $ErrorActionPreference = 'Continue'
      ${command}
      if ($ErrorActionPreference -ne 'Continue') { throw 'parent preferences changed' }
      $undefinedParentVariable
      Write-Output 'PARENT_ALIVE'
    `], { env: { ...process.env, USERPROFILE: root }, encoding: 'utf8' })
    expect(result.status, result.stderr).toBe(0)
    expect(result.stdout).toContain('PARENT_ALIVE')
    return `${result.stdout}\n${result.stderr}`
  }

  it('stops without creating the directory or changing the parent scope', () => {
    const { root } = sandbox()
    expect(run(root)).toContain('未找到 Codex 配置目录')
    expect(readdirSync(root)).toEqual(['bin'])
  })

  it('stops when Codex is running', () => {
    const { root, dir } = sandbox()
    mkdirSync(dir)
    expect(run(root, { running: true })).toContain('仍在运行')
    expect(readdirSync(dir)).toEqual([])
  })

  it.each([[], ['config.toml'], ['auth.json'], ['config.toml', 'auth.json']].map((existing) => ({ existing })))('writes UTF-8 without BOM and keeps unique backups: $existing', ({ existing }) => {
    const { root, dir } = sandbox()
    mkdirSync(dir)
    for (const file of existing) writeFileSync(join(dir, file), `old ${file}`)
    expect(run(root)).toContain('配置完成')
    expect(readFileSync(join(dir, 'config.toml'), 'utf8')).toBe(config)
    expect(readFileSync(join(dir, 'auth.json'), 'utf8')).toBe(auth)
    expect(run(root)).toContain('配置完成')
    expect(readdirSync(dir).filter((name) => name.includes('.bak-'))).toHaveLength(existing.length + 2)
    expect(readdirSync(dir).some((name) => name.startsWith('.sub2api-'))).toBe(false)
  })

  it.each([[], ['config.toml'], ['auth.json'], ['config.toml', 'auth.json']].map((existing) => ({ existing })))('rolls back a failed second replacement: $existing', ({ existing }) => {
    const { root, dir } = sandbox()
    mkdirSync(dir)
    for (const file of existing) writeFileSync(join(dir, file), `old ${file}`)
    const output = run(root, { failReplace: true })
    expect(output).toContain('simulated auth replacement failure')
    expect(output).toContain('已恢复')
    expect(output).not.toContain('配置完成')
    expect(readdirSync(dir).filter((name) => !name.includes('.bak-')).sort()).toEqual(existing.slice().sort())
    for (const file of existing) expect(readFileSync(join(dir, file), 'utf8')).toBe(`old ${file}`)
  })

  it('rejects invalid auth and preserves literal special characters', () => {
    const { root, dir } = sandbox()
    mkdirSync(dir)
    expect(run(root, { auth: '{invalid' })).toContain('auth.json JSON 解析校验失败')
    expect(readdirSync(dir)).toEqual([])
    const value = '\'"$HOME`echo bad`$(echo bad)\\\n\'@\n中文'
    const configContent = `model = "${escapeTomlBasicString(value)}"\n`
    const authContent = JSON.stringify({ OPENAI_API_KEY: value }, null, 2)
    expect(run(root, { config: configContent, auth: authContent })).toContain('配置完成')
    expect(readFileSync(join(dir, 'config.toml'), 'utf8')).toBe(configContent)
    expect(readFileSync(join(dir, 'auth.json'), 'utf8')).toBe(authContent)
  })
})
