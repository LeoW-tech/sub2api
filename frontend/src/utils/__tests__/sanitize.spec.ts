import { describe, expect, it } from 'vitest'

import { sanitizeSvg } from '@/utils/sanitize'

describe('sanitizeSvg', () => {
  it('移除内联事件处理器，同时保留安全的 svg 结构', () => {
    const input = '<svg viewBox="0 0 24 24"><g onclick="alert(1)"><path d="M0 0h24v24z" /></g></svg>'

    const output = sanitizeSvg(input)

    expect(output).toContain('<svg')
    expect(output).toContain('<path')
    expect(output).not.toContain('onclick')
  })
})
