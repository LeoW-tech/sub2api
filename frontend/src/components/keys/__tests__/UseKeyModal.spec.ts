import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn().mockResolvedValue(true)
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
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.installCodexTitle')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.quitCodexTitle')
    expect(wrapper.text()).toContain('keys.useKeyModal.openai.copyCommand')
    expect(wrapper.text()).not.toContain('keys.useKeyModal.cliTabs.codexCliWs')
    expect(wrapper.text()).not.toContain('keys.useKeyModal.cliTabs.opencode')
  })

  it('renders macOS one-click command with safety checks and success contact hint', () => {
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

    const command = wrapper.find('[data-testid="codex-one-click-command"]').text()

    expect(command).toContain('$HOME/.codex')
    expect(command).toContain('[ ! -d "$CODEX_DIR" ]')
    expect(command).toContain('pgrep -if')
    expect(command).toContain('$CONFIG_FILE.sub2api.bak-')
    expect(command).toContain('$AUTH_FILE.sub2api.bak-')
    expect(command).toContain('model = "gpt-5.5"')
    expect(command).toContain('"OPENAI_API_KEY": "sk-test"')
    expect(command).toContain('open "$CODEX_DIR"')
    expect(command).toContain('配置成功')
    expect(command).toContain('现在可以打开 Codex 开始使用')
    expect(command).toContain('网页右上角联系客服咨询')
  })

  it('renders Windows PowerShell one-click command with safety checks', () => {
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

    const command = wrapper.find('[data-testid="codex-one-click-command"]').text()

    expect(wrapper.text()).toContain('keys.useKeyModal.openai.detectedWindows')
    expect(command).toContain('$env:USERPROFILE')
    expect(command).toContain('Test-Path $codexDir')
    expect(command).toContain('Get-Process')
    expect(command).toContain('$configFile.sub2api.bak-')
    expect(command).toContain('$authFile.sub2api.bak-')
    expect(command).toContain('model = "gpt-5.5"')
    expect(command).toContain('"OPENAI_API_KEY": "sk-test"')
    expect(command).toContain('Invoke-Item $codexDir')
  })

  it('renders copyable professional directory hints and download buttons', () => {
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
    expect(wrapper.findAll('[data-testid="copy-snippet-button"]').length).toBeGreaterThanOrEqual(4)
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

})
