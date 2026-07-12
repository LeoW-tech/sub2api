import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

const clipboardMock = vi.hoisted(() => ({
  copyToClipboard: vi.fn().mockResolvedValue(true)
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: clipboardMock.copyToClipboard
  })
}))

import UseKeyModal from '../UseKeyModal.vue'

describe('UseKeyModal', () => {
  const originalUserAgent = window.navigator.userAgent

  const setUserAgent = (value: string) => {
    Object.defineProperty(window.navigator, 'userAgent', {
      value,
      configurable: true
    })
  }

  afterEach(() => {
    setUserAgent(originalUserAgent)
    clipboardMock.copyToClipboard.mockClear()
    vi.restoreAllMocks()
  })

  it('renders GPT-5.5 and goals feature in OpenAI Codex config', () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1',
        platform: 'openai'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    const codeBlocks = wrapper.findAll('pre code').map((code) => code.text())
    const configToml = codeBlocks.find((content) => content.includes('model_provider = "OpenAI"'))

    expect(configToml).toBeDefined()
    expect(configToml).toContain('model = "gpt-5.5"')
    expect(configToml).toContain('review_model = "gpt-5.5"')
    expect(configToml).not.toContain('model = "gpt-5.4"')
    expect(configToml).not.toContain('model_context_window')
    expect(configToml).not.toContain('model_auto_compact_token_limit')
    expect(configToml).toContain('[features]\ngoals = true')
  })

  it('renders beginner and professional methods for Codex CLI only', () => {
    setUserAgent('Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/537.36')

    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1',
        platform: 'openai'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    expect(wrapper.text()).toContain('keys.useKeyModal.openai.beginner.title')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.professional.title')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.detectedMac')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.steps.install')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.steps.quit')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.copyCommand')
    expect(wrapper.text()).not.toContain('keys.useKeyModal.cliTabs.codexCliWs')
    expect(wrapper.text()).not.toContain('keys.useKeyModal.cliTabs.opencode')

    const downloadLinks = wrapper.findAll('a[target="_blank"]')
    const hrefs = downloadLinks.map((link) => link.attributes('href'))
    expect(hrefs).toContain('https://chatgpt.com/zh-Hans-CN/download/')
    expect(hrefs).toContain('https://t3.znas.cn/H0ogPWjF07')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.backupDownload')
    expect(wrapper.text()).not.toContain('keys.useKeyModal.openai.copyDownloadLink')
  })

  it('renders macOS one-click command with safety checks and success contact hint', async () => {
    setUserAgent('Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/537.36')

    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1',
        platform: 'openai'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    const commandButton = wrapper.find('[data-testid="copy-codex-one-click-command"]')

    expect(wrapper.text()).toContain('keys.useKeyModal.openai.steps.install')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.steps.quit')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.steps.openTerminal')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.steps.copyCommand')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.quitCodexDescriptionPrefix')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.steps.openTerminal')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.macTerminalStep2')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.terminalPreviewCaption')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.commandStep2Mac')
    expect(commandButton.exists()).toBe(true)
    expect(wrapper.find('[data-testid="codex-one-click-command"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('CODEX_DIR="$HOME/.codex"')

    await commandButton.trigger('click')
    expect(clipboardMock.copyToClipboard).toHaveBeenCalledTimes(1)
    const copiedCommand = String(clipboardMock.copyToClipboard.mock.calls[0]?.[0] ?? '')
    expect(copiedCommand).toContain('https://chatgpt.com/zh-Hans-CN/download/')
    expect(copiedCommand).toContain('https://t3.znas.cn/H0ogPWjF07')
    expect(copiedCommand).not.toContain('https://chatgpt.com/codex/for-work/')
  })

  it('renders Windows PowerShell one-click command with safety checks', async () => {
    setUserAgent('Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36')

    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1',
        platform: 'openai'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    expect(wrapper.text()).toContain('keys.useKeyModal.openai.detectedWindows')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.commandStep2Windows')
    expect(wrapper.find('[data-testid="copy-codex-one-click-command"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="codex-one-click-command"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('$codexDir = Join-Path')

    await wrapper.find('[data-testid="copy-codex-one-click-command"]').trigger('click')
    expect(clipboardMock.copyToClipboard).toHaveBeenCalledTimes(1)
    const copiedCommand = String(clipboardMock.copyToClipboard.mock.calls[0]?.[0] ?? '')
    expect(copiedCommand).toContain('https://chatgpt.com/zh-Hans-CN/download/')
    expect(copiedCommand).toContain('https://t3.znas.cn/H0ogPWjF07')
    expect(copiedCommand).not.toContain('https://chatgpt.com/codex/for-work/')
  })

  it('renders professional setup as directory, config.toml, and auth.json steps', () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1',
        platform: 'openai'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    expect(wrapper.text()).toContain('~/.codex')
    expect(wrapper.text()).toContain('%USERPROFILE%\\.codex')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.professional.steps.openDir')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.professional.steps.configToml')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.professional.steps.authJson')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.professional.configTomlDownloadHint')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.professional.authJsonDownloadHint')

    const configSection = wrapper.find('[data-testid="professional-config-toml-step"]')
    const authSection = wrapper.find('[data-testid="professional-auth-json-step"]')
    expect(configSection.text().indexOf('keys.useKeyModal.openai.professional.configTomlDownloadHint'))
      .toBeLessThan(configSection.text().indexOf('model_provider = "OpenAI"'))
    expect(authSection.text().indexOf('keys.useKeyModal.openai.professional.authJsonDownloadHint'))
      .toBeLessThan(authSection.text().indexOf('"OPENAI_API_KEY": "sk-test"'))

    expect(wrapper.findAll('[data-testid="copy-snippet-button"]').length).toBeGreaterThanOrEqual(3)
    expect(wrapper.findAll('[data-testid="download-config-button"]')).toHaveLength(2)
  })

  it('hides OpenAI Codex WebSocket and OpenCode beginner-unfriendly tabs', () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1',
        platform: 'openai'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    expect(wrapper.text()).not.toContain('keys.useKeyModal.cliTabs.codexCliWs')
    expect(wrapper.text()).not.toContain('keys.useKeyModal.cliTabs.opencode')
    expect(wrapper.text()).not.toContain('supports_websockets = true')
    expect(wrapper.text()).not.toContain('GPT-5.4 Mini')
  })


  it('renders Claude Fable 5 OpenCode config with adaptive thinking', async () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1',
        platform: 'antigravity'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    const opencodeTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.opencode')
    )

    expect(opencodeTab).toBeDefined()
    await opencodeTab!.trigger('click')
    await nextTick()

    const claudeConfig = wrapper.findAll('pre code')
      .map((code) => code.text())
      .find((content) => content.includes('"antigravity-claude"'))

    expect(claudeConfig).toBeDefined()
    const parsed = JSON.parse(claudeConfig!)
    const fable = parsed.provider['antigravity-claude'].models['claude-fable-5']

    expect(fable.name).toBe('Claude Fable 5')
    expect(fable.limit).toEqual({ context: 1048576, output: 128000 })
    expect(fable.options.thinking).toEqual({ type: 'adaptive' })
    expect(fable.options.thinking).not.toHaveProperty('budgetTokens')
  })

})
