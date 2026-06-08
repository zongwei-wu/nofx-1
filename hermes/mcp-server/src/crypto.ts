import { createPublicKey, publicEncrypt, randomBytes, createCipheriv } from 'node:crypto'

export interface EncryptedPayload {
  wrappedKey: string
  iv: string
  ciphertext: string
  aad?: string
  kid?: string
  ts?: number
}

let cachedPublicKeyPem: string | null = null

function arrayBufferToBase64Url(buffer: Buffer): string {
  return buffer
    .toString('base64')
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=/g, '')
}

function base64UrlToBuffer(value: string): Buffer {
  const padded = value + '='.repeat((4 - (value.length % 4)) % 4)
  return Buffer.from(padded.replace(/-/g, '+').replace(/_/g, '/'), 'base64')
}

export async function fetchPublicKey(apiBase: string): Promise<string> {
  if (cachedPublicKeyPem) return cachedPublicKeyPem
  const res = await fetch(`${apiBase}/api/crypto/public-key`)
  if (!res.ok) {
    throw new Error(`Failed to fetch public key: ${res.status}`)
  }
  const data = (await res.json()) as { public_key: string }
  cachedPublicKeyPem = data.public_key
  return cachedPublicKeyPem
}

export async function encryptSensitiveData(
  apiBase: string,
  plaintext: string,
  userId?: string
): Promise<EncryptedPayload> {
  const publicKeyPem = await fetchPublicKey(apiBase)
  const publicKey = createPublicKey(publicKeyPem)

  const aesKey = randomBytes(32)
  const iv = randomBytes(12)
  const ts = Math.floor(Date.now() / 1000)
  const aadObject = {
    userId: userId || '',
    sessionId: '',
    ts,
    purpose: 'sensitive_data_encryption',
  }
  const aadString = JSON.stringify(aadObject)
  const aadBytes = Buffer.from(aadString, 'utf8')

  const cipher = createCipheriv('aes-256-gcm', aesKey, iv)
  cipher.setAAD(aadBytes)
  const encrypted = Buffer.concat([
    cipher.update(Buffer.from(plaintext, 'utf8')),
    cipher.final(),
  ])
  const tag = cipher.getAuthTag()
  const ciphertext = Buffer.concat([encrypted, tag])

  const wrappedKey = publicEncrypt(
    {
      key: publicKey,
      padding: 1, // RSA_PKCS1_OAEP_PADDING
      oaepHash: 'sha256',
    },
    aesKey
  )

  return {
    wrappedKey: arrayBufferToBase64Url(wrappedKey),
    iv: arrayBufferToBase64Url(iv),
    ciphertext: arrayBufferToBase64Url(ciphertext),
    aad: arrayBufferToBase64Url(aadBytes),
    ts,
  }
}

export { base64UrlToBuffer }
