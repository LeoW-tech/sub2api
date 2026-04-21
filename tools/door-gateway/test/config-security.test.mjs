import test from 'node:test'
import assert from 'node:assert/strict'

import { parseSourceConfigText } from '../src/config.mjs'

test('parseSourceConfigText rejects deeply nested yaml with a parse error', () => {
  const payload = '['.repeat(5000) + '1' + ']'.repeat(5000)

  assert.throws(
    () => parseSourceConfigText(payload, 'deep-nesting'),
    /failed to parse source config deep-nesting/i
  )
})
