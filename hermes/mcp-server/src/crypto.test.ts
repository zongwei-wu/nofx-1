import { describe, expect, it } from 'vitest'
import { base64UrlToBuffer } from './crypto.js'

describe('crypto', () => {
  it('base64UrlToBuffer roundtrip', () => {
    const original = Buffer.from('hello')
    const b64url = original
      .toString('base64')
      .replace(/\+/g, '-')
      .replace(/\//g, '_')
      .replace(/=/g, '')
    expect(base64UrlToBuffer(b64url).toString()).toBe('hello')
  })
})
