import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

const { copyToClipboardMock } = vi.hoisted(() => ({
  copyToClipboardMock: vi.fn().mockResolvedValue(true)
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: copyToClipboardMock
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
    copyToClipboardMock.mockClear()
    vi.restoreAllMocks()
  })
  it('renders Grok Build and OpenCode setup for Grok groups', async () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-grok-test',
        baseUrl: 'https://example.com/v1',
        platform: 'grok'
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

    const grokTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.grokCli')
    )
    expect(grokTab).toBeDefined()

    const allCode = wrapper.findAll('pre code').map((code) => code.text()).join('\n')
    expect(allCode).toContain('GROK_MODELS_BASE_URL')
    expect(allCode).toContain('XAI_API_KEY')
    expect(allCode).toContain('[model."grok-4.5"]')
    expect(allCode).toContain('[model."grok-build-0.1"]')
    expect(allCode).toContain('[model."grok-4.20-multi-agent-0309"]')
    expect(allCode).toContain('[model."grok-4.3"]')
    expect(allCode).toContain('default = "grok-4.5"')
    expect(allCode).toContain('models_base_url = "https://example.com/v1"')
    expect(allCode).toContain('models_list_url = "https://example.com/v1/models"')
    expect(allCode).toContain('xai_api_base_url = "https://example.com/v1"')
    expect(allCode).toContain('cli_chat_proxy_base_url = "https://example.com/v1"')
    expect(allCode).toContain('preferred_method = "api_key"')
    expect(allCode).toContain('image_description = "grok-4.5"')
    expect(allCode).toContain('auto_compact_threshold_percent = 80')
    expect(allCode).toContain('image_gen = true')
    expect(allCode).toContain('video_gen = true')
    expect(allCode).toContain('image_gen_model_override = "grok-imagine-image-quality"')
    expect(allCode).toContain('image_edit_model_override = "grok-imagine-edit"')
    expect(allCode).toContain('env_key = "XAI_API_KEY"')
    expect(allCode).toContain('Keep api_backend = "responses" on every model entry.')
    expect(allCode).toContain('grok-imagine-image')
    expect(allCode).toContain('grok-imagine-edit')
    expect(allCode).toMatch(/\[model\."grok-4\.5"\][\s\S]*?context_window = 500000/)
    expect(allCode).toMatch(/\[model\."grok-build-0\.1"\][\s\S]*?context_window = 256000/)
    // Prefer env_key; hardcode api_key only as commented alternative
    expect(allCode).not.toMatch(/^api_key = "sk-grok-test"$/m)

    const modelBlocks = allCode
      .split(/(?=^\[model\.)/m)
      .filter((block) => block.startsWith('[model."'))
    expect(modelBlocks.length).toBeGreaterThanOrEqual(4)
    for (const block of modelBlocks) {
      if (block.includes('# [model.')) continue
      expect(block).toContain('api_backend = "responses"')
    }

    const windowsTab = wrapper.findAll('button').find(
      (button) => button.text().trim() === 'Windows'
    )
    expect(windowsTab).toBeDefined()
    await windowsTab!.trigger('click')
    await nextTick()
    expect(wrapper.text().toLowerCase()).toContain('%userprofile%\\.grok\\config.toml')

    const opencodeTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.opencode')
    )
    expect(opencodeTab).toBeDefined()
    await opencodeTab!.trigger('click')
    await nextTick()

    const parsed = JSON.parse(wrapper.find('pre code').text())
    expect(parsed.provider.grok.npm).toBe('@ai-sdk/openai-compatible')
    expect(parsed.provider.grok.name).toBe('Grok via Sub2API')
    expect(parsed.provider.grok.options).toEqual({
      baseURL: 'https://example.com/v1',
      apiKey: 'sk-grok-test'
    })
    expect(parsed.provider.grok.models['grok-4.5']).toBeDefined()
    expect(parsed.provider.grok.models['grok-4.5'].limit.context).toBe(500000)
    expect(parsed.provider.grok.models['grok-build-0.1']).toBeDefined()
    expect(parsed.provider.grok.models['grok-4.20-multi-agent-0309']).toBeDefined()
    expect(parsed.provider.grok.models['grok-composer-2.5-fast']).toBeDefined()
    expect(parsed.provider.grok.models['gpt-5.6']).toBeUndefined()
  })

  it('renders copyable Claude Code setup through the Grok Messages gateway', async () => {
    copyToClipboardMock.mockClear()
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-grok-claude-test',
        baseUrl: 'https://example.com/v1',
        platform: 'grok'
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

    const claudeTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.claudeCode')
    )
    expect(claudeTab).toBeDefined()
    await claudeTab!.trigger('click')
    await nextTick()

    let codeBlocks = wrapper.findAll('pre code').map((code) => code.text())
    expect(codeBlocks.join('\n')).toContain('ANTHROPIC_BASE_URL="https://example.com"')
    expect(codeBlocks.join('\n')).toContain('ANTHROPIC_AUTH_TOKEN="sk-grok-claude-test"')
    const unixConfig = codeBlocks.find((content) => content.startsWith('export ANTHROPIC_BASE_URL'))
    expect(unixConfig).toBeDefined()
    for (const name of [
      'ANTHROPIC_MODEL',
      'ANTHROPIC_DEFAULT_OPUS_MODEL',
      'ANTHROPIC_DEFAULT_SONNET_MODEL',
      'ANTHROPIC_DEFAULT_HAIKU_MODEL',
      'ANTHROPIC_DEFAULT_FABLE_MODEL',
      'CLAUDE_CODE_SUBAGENT_MODEL'
    ]) {
      expect(unixConfig).toContain(`export ${name}="grok-4.5"`)
    }
    const settingsConfig = codeBlocks.find((content) => content.includes('"$schema"'))
    expect(settingsConfig).toBeDefined()
    const parsedSettings = JSON.parse(settingsConfig!)
    expect(parsedSettings.$schema).toBe('https://json.schemastore.org/claude-code-settings.json')
    expect(parsedSettings.env.ANTHROPIC_MODEL).toBe('grok-4.5')
    expect(wrapper.text()).toContain('keys.useKeyModal.claudeSettingsHint')
    expect(wrapper.text()).toContain('keys.useKeyModal.grok.claudeNote')
    expect(wrapper.find('nav[aria-label="Client"]').classes()).toContain('min-w-max')
    expect(wrapper.find('nav[aria-label="Client"]').element.parentElement?.classList.contains('overflow-x-auto')).toBe(true)

    const cmdTab = wrapper.findAll('button').find(
      (button) => button.text().trim() === 'Windows CMD'
    )
    expect(cmdTab).toBeDefined()
    await cmdTab!.trigger('click')
    await nextTick()

    codeBlocks = wrapper.findAll('pre code').map((code) => code.text())
    expect(codeBlocks.join('\n')).toContain('set ANTHROPIC_MODEL=grok-4.5')
    expect(codeBlocks.join('\n')).toContain('set ANTHROPIC_DEFAULT_FABLE_MODEL=grok-4.5')
    expect(codeBlocks.join('\n')).toContain('set CLAUDE_CODE_SUBAGENT_MODEL=grok-4.5')

    const powershellTab = wrapper.findAll('button').find(
      (button) => button.text().trim() === 'PowerShell'
    )
    expect(powershellTab).toBeDefined()
    await powershellTab!.trigger('click')
    await nextTick()

    codeBlocks = wrapper.findAll('pre code').map((code) => code.text())
    expect(codeBlocks.join('\n')).toContain('$env:ANTHROPIC_BASE_URL="https://example.com"')
    expect(codeBlocks.join('\n')).toContain('$env:ANTHROPIC_MODEL="grok-4.5"')
    expect(codeBlocks.join('\n')).toContain('$env:ANTHROPIC_DEFAULT_FABLE_MODEL="grok-4.5"')
    expect(codeBlocks.join('\n')).toContain('$env:CLAUDE_CODE_SUBAGENT_MODEL="grok-4.5"')
    expect(wrapper.text()).toContain('%USERPROFILE%\\.claude\\settings.json')

    const copyButton = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.copy')
    )
    expect(copyButton).toBeDefined()
    await copyButton!.trigger('click')
    expect(copyToClipboardMock).toHaveBeenCalledWith(
      expect.stringContaining('ANTHROPIC_AUTH_TOKEN="sk-grok-claude-test"'),
      'keys.copied'
    )
  })

  it('renders Codex custom provider setup through the Grok Responses gateway', async () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-grok-codex-test',
        baseUrl: 'https://example.com/v1',
        platform: 'grok'
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

    const codexTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.codexCli')
    )
    expect(codexTab).toBeDefined()
    await codexTab!.trigger('click')
    await nextTick()

    let codeBlocks = wrapper.findAll('pre code').map((code) => code.text())
    const configToml = codeBlocks.find((content) => content.includes('[model_providers.sub2api]'))
    expect(configToml).toBeDefined()
    expect(configToml).toContain('model_provider = "sub2api"')
    expect(configToml).toContain('model = "grok-4.5"')
    expect(configToml).toContain('base_url = "https://example.com/v1"')
    expect(configToml).toContain('env_key = "SUB2API_API_KEY"')
    expect(configToml).toContain('wire_api = "responses"')
    // API-key provider: Codex must not require a ChatGPT OAuth login.
    expect(configToml).toContain('requires_openai_auth = false')
    expect(configToml).toContain('supports_websockets = false')
    expect(configToml).toContain('grok-4.20-multi-agent-0309 (text / web_search)')
    expect(configToml).toContain('grok-imagine-image')
    expect(configToml).toContain('grok-imagine-video')
    // Hardcoded bearer is only a commented fallback when env cannot be set.
    expect(configToml).toMatch(/# experimental_bearer_token = "sk-grok-codex-test"/)
    expect(configToml).not.toContain('supports_websockets = true')
    expect(configToml).not.toContain('responses_websockets_v2')
    expect(wrapper.text()).not.toContain('auth.json')
    expect(codeBlocks.join('\n')).toContain('SUB2API_API_KEY')

    const windowsTab = wrapper.findAll('button').find(
      (button) => button.text().trim() === 'Windows'
    )
    expect(windowsTab).toBeDefined()
    await windowsTab!.trigger('click')
    await nextTick()

    codeBlocks = wrapper.findAll('pre code').map((code) => code.text())
    expect(wrapper.text().toLowerCase()).toContain('%userprofile%\\.codex\\config.toml'.toLowerCase())
    expect(codeBlocks.join('\n')).toContain('experimental_bearer_token = "sk-grok-codex-test"')
  })

  it('keeps legacy OpenAI Codex config as the default', () => {
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
    expect(configToml).toBe([
      'model_provider = "OpenAI"',
      'model = "gpt-5.6-sol"',
      'model_reasoning_effort = "xhigh"',
      'approval_policy = "never"',
      'sandbox_mode = "danger-full-access"',
      '',
      '[model_providers.OpenAI]',
      'name = "OpenAI"',
      'base_url = "https://example.com/v1"',
      'wire_api = "responses"',
      'requires_openai_auth = true',
      '',
      '[features]',
      'goals = true'
    ].join('\n'))
    expect(configToml?.match(/requires_openai_auth/g)).toHaveLength(1)
    expect(codeBlocks).toContain('{\n  "OPENAI_API_KEY": "sk-test"\n}')
    expect(wrapper.text()).toContain('auth.json')
    expect(wrapper.find('[data-testid="codex-api-key-restart-notice"]').exists()).toBe(false)
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
    expect(wrapper.find('nav[aria-label="Client"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="codex-auth-mode-legacy"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('keys.useKeyModal.cliTabs.opencode')
    expect(wrapper.text()).toContain('keys.useKeyModal.cliTabs.codexCliWs')

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
    expect(copyToClipboardMock).toHaveBeenCalledTimes(1)
    const copiedCommand = String(copyToClipboardMock.mock.calls[0]?.[0] ?? '')
    expect(copiedCommand).toContain('https://chatgpt.com/zh-Hans-CN/download/')
    expect(copiedCommand).toContain('https://t3.znas.cn/H0ogPWjF07')
    expect(copiedCommand).toContain('set -Eeuo pipefail')
    expect(copiedCommand).toContain('cat > "$CONFIG_FILE"')
    expect(copiedCommand).toContain('cat > "$AUTH_FILE"')
    expect(copiedCommand).toContain('approval_policy = "never"')
    expect(copiedCommand).toContain('sandbox_mode = "danger-full-access"')
    expect(copiedCommand).toContain('✅ Codex完成接入！')
    expect(copiedCommand).not.toContain('配置成功！Codex CLI 已完成接入')
    expect(copiedCommand).not.toContain('review_model =')
    expect(copiedCommand).not.toContain('disable_response_storage')
    expect(copiedCommand).not.toContain('network_access')
    expect(copiedCommand).not.toContain('sandbox_workspace_write')
    expect(copiedCommand).not.toContain('windows_wsl_setup_acknowledged')
    expect(copiedCommand).not.toContain('https://chatgpt.com/codex/for-work/')

    const configToml = wrapper.find('[data-testid="professional-config-toml-step"] pre code').text()
    expect(copiedCommand).toContain(configToml)
    expect(copiedCommand).toContain('{\n  "OPENAI_API_KEY": "sk-test"\n}')
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
    expect(copyToClipboardMock).toHaveBeenCalledTimes(1)
    const copiedCommand = String(copyToClipboardMock.mock.calls[0]?.[0] ?? '')
    expect(copiedCommand).toContain('https://chatgpt.com/zh-Hans-CN/download/')
    expect(copiedCommand).toContain('https://t3.znas.cn/H0ogPWjF07')
    expect(copiedCommand).toContain("$ErrorActionPreference = 'Stop'")
    expect(copiedCommand).toContain('Set-StrictMode -Version Latest')
    expect(copiedCommand).toContain('Set-Content -Path $configFile')
    expect(copiedCommand).toContain('Set-Content -Path $authFile')
    expect(copiedCommand).toContain('approval_policy = "never"')
    expect(copiedCommand).toContain('sandbox_mode = "danger-full-access"')
    expect(copiedCommand).toContain('✅ Codex完成接入！')
    expect(copiedCommand).not.toContain('配置成功！Codex CLI 已完成接入')
    expect(copiedCommand).not.toContain('review_model =')
    expect(copiedCommand).not.toContain('disable_response_storage')
    expect(copiedCommand).not.toContain('network_access')
    expect(copiedCommand).not.toContain('sandbox_workspace_write')
    expect(copiedCommand).not.toContain('windows_wsl_setup_acknowledged')
    expect(copiedCommand).not.toContain('https://chatgpt.com/codex/for-work/')

    const configToml = wrapper.find('[data-testid="professional-config-toml-step"] pre code').text()
    expect(copiedCommand).toContain(configToml)
    expect(copiedCommand).toContain('{\n  "OPENAI_API_KEY": "sk-test"\n}')
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

})
