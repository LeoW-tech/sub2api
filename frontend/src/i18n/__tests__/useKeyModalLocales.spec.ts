import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

const openAICodexKeys = [
  'keys.useKeyModal.download',
  'keys.useKeyModal.openai.description',
  'keys.useKeyModal.openai.detectedMac',
  'keys.useKeyModal.openai.detectedWindows',
  'keys.useKeyModal.openai.detectedOther',
  'keys.useKeyModal.openai.unsupportedSystem',
  'keys.useKeyModal.openai.installCodexDescription',
  'keys.useKeyModal.openai.downloadCodex',
  'keys.useKeyModal.openai.backupDownload',
  'keys.useKeyModal.openai.quitCodexDescriptionPrefix',
  'keys.useKeyModal.openai.quitCodexDescription',
  'keys.useKeyModal.openai.openPowerShellTitle',
  'keys.useKeyModal.openai.macTerminalStep1',
  'keys.useKeyModal.openai.macTerminalStep2',
  'keys.useKeyModal.openai.windowsTerminalStep1',
  'keys.useKeyModal.openai.windowsTerminalStep2',
  'keys.useKeyModal.openai.terminalPreviewCaption',
  'keys.useKeyModal.openai.powerShellPreviewCaption',
  'keys.useKeyModal.openai.copyInput',
  'keys.useKeyModal.openai.copyCommand',
  'keys.useKeyModal.openai.commandStep2Mac',
  'keys.useKeyModal.openai.commandStep2Windows',
  'keys.useKeyModal.openai.commandStep3',
  'keys.useKeyModal.openai.steps.install',
  'keys.useKeyModal.openai.steps.quit',
  'keys.useKeyModal.openai.steps.openTerminal',
  'keys.useKeyModal.openai.steps.copyCommand',
  'keys.useKeyModal.openai.beginner.title',
  'keys.useKeyModal.openai.professional.title',
  'keys.useKeyModal.openai.professional.description',
  'keys.useKeyModal.openai.professional.macOpenDir',
  'keys.useKeyModal.openai.professional.windowsOpenDir',
  'keys.useKeyModal.openai.professional.configTomlDownloadHint',
  'keys.useKeyModal.openai.professional.configTomlCopyHint',
  'keys.useKeyModal.openai.professional.authJsonDownloadHint',
  'keys.useKeyModal.openai.professional.steps.openDir',
  'keys.useKeyModal.openai.professional.steps.configToml',
  'keys.useKeyModal.openai.professional.steps.authJson'
]

function getMessage(locale: Record<string, unknown>, path: string): unknown {
  return path.split('.').reduce<unknown>((value, key) => {
    if (!value || typeof value !== 'object') return undefined
    return (value as Record<string, unknown>)[key]
  }, locale)
}

describe('UseKeyModal OpenAI Codex locales', () => {
  it.each([
    ['zh', zh],
    ['en', en]
  ] as const)('defines every rendered OpenAI Codex message in %s', (_locale, messages) => {
    for (const key of openAICodexKeys) {
      expect(getMessage(messages, key), key).toEqual(expect.any(String))
    }
  })

  it('keeps the two configuration modes explicit in Chinese', () => {
    expect(zh.keys.useKeyModal.openai.beginner.title).toBe('方法一：小白一键配置')
    expect(zh.keys.useKeyModal.openai.professional.title).toBe('方法二：专业人士使用')
  })

  it('keeps the Codex download links and manual download action in Chinese', () => {
    expect(zh.keys.useKeyModal.openai.downloadCodex).toBe('地址 1：CODEX 官方下载页面（直接点击打开）')
    expect(zh.keys.useKeyModal.openai.backupDownload).toBe('地址 2：备用网盘（如果地址 1 官方页面无法打开，请点此下载）')
    expect(zh.keys.useKeyModal.download).toBe('点击下载')
  })
})
