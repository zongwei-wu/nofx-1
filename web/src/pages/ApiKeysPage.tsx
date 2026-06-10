import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { Container } from '../components/Container'
import { Key, Copy, Trash2, Plus, AlertTriangle, Check, X } from 'lucide-react'

interface ApiKeyRecord {
  id: string
  key_prefix: string
  name: string
  last_used: string | null
  expires_at: string | null
  created_at: string
}

export default function ApiKeysPage() {
  const { token } = useAuth()
  const [keys, setKeys] = useState<ApiKeyRecord[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [creating, setCreating] = useState(false)
  const [newKeyName, setNewKeyName] = useState('')
  const [newKeyPlaintext, setNewKeyPlaintext] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [deleting, setDeleting] = useState<string | null>(null)

  const loadKeys = useCallback(async () => {
    if (!token) return
    try {
      setLoading(true)
      setError('')
      const res = await fetch('/api/api-keys', {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (!res.ok) throw new Error('获取失败')
      const data = await res.json()
      setKeys(Array.isArray(data) ? data : [])
    } catch (e: any) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { loadKeys() }, [loadKeys])

  const handleCreate = async () => {
    if (!token) return
    try {
      setCreating(true)
      setError('')
      const res = await fetch('/api/api-keys', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ name: newKeyName || '' }),
      })
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw new Error(body.error || '创建失败')
      }
      const data = await res.json()
      setNewKeyPlaintext(data.api_key)
      setNewKeyName('')
      await loadKeys()
    } catch (e: any) {
      setError(e.message)
    } finally {
      setCreating(false)
    }
  }

  const handleCopy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text)
    } catch {
      const el = document.createElement('textarea')
      el.value = text
      document.body.appendChild(el)
      el.select()
      document.execCommand('copy')
      document.body.removeChild(el)
    }
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleDelete = async (id: string) => {
    if (!token) return
    if (!confirm('确定要撤销这个 API Key 吗？撤销后立即失效。')) return
    try {
      setDeleting(id)
      const res = await fetch(`/api/api-keys/${id}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      })
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw new Error(body.error || '删除失败')
      }
      await loadKeys()
    } catch (e: any) {
      setError(e.message)
    } finally {
      setDeleting(null)
    }
  }

  if (!token) {
    return (
      <Container className="py-6">
        <div className="max-w-3xl mx-auto text-center py-16">
          <Key className="w-12 h-12 mx-auto mb-3" style={{ color: '#2B3139' }} />
          <p style={{ color: '#848E9C', fontSize: '18px' }}>请先登录后查看 API Keys</p>
        </div>
      </Container>
    )
  }

  return (
    <Container className="py-6">
      <div className="max-w-3xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-2xl font-bold mb-2" style={{ color: '#EAECEF' }}>
            <Key className="inline-block w-6 h-6 mr-2" style={{ color: 'var(--brand-yellow)' }} />
            API Keys
          </h1>
          <p style={{ color: '#848E9C' }}>
            生成 API Key 供外部 Hermes 服务调用交易所网关（/api/gateway/*）。Key 仅在创建时展示一次，请立即复制保存。
          </p>
        </div>

        {/* Error */}
        {error && (
          <div
            className="mb-4 p-3 rounded-lg flex items-center gap-2"
            style={{ background: 'rgba(246, 70, 93, 0.1)', border: '1px solid rgba(246, 70, 93, 0.3)', color: '#F6465D' }}
          >
            <AlertTriangle className="w-4 h-4 flex-shrink-0" />
            <span className="flex-1 text-sm">{error}</span>
            <button onClick={() => setError('')} className="flex-shrink-0"><X className="w-4 h-4" /></button>
          </div>
        )}

        {/* New Key Display (one-time) */}
        {newKeyPlaintext && (
          <div
            className="mb-6 p-4 rounded-lg"
            style={{ background: 'rgba(14, 203, 129, 0.1)', border: '1px solid rgba(14, 203, 129, 0.3)' }}
          >
            <div className="flex items-center gap-2 mb-2">
              <Check className="w-5 h-5" style={{ color: '#0ECB81' }} />
              <span className="font-bold" style={{ color: '#0ECB81' }}>API Key 创建成功！</span>
            </div>
            <p className="text-sm mb-3" style={{ color: '#848E9C' }}>这是唯一一次展示完整 Key 的机会，请立即复制保存。</p>
            <div className="flex items-center gap-2">
              <code className="flex-1 p-2 rounded font-mono text-sm break-all" style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}>
                {newKeyPlaintext}
              </code>
              <button onClick={() => handleCopy(newKeyPlaintext)} className="px-3 py-2 rounded-lg font-bold text-sm flex items-center gap-1" style={{ background: '#0ECB81', color: '#0B0E11' }}>
                {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
                {copied ? '已复制' : '复制'}
              </button>
            </div>
            <button onClick={() => setNewKeyPlaintext(null)} className="mt-3 text-sm underline" style={{ color: '#848E9C' }}>
              我已保存，关闭提示
            </button>
          </div>
        )}

        {/* Create */}
        <div className="mb-6 p-4 rounded-lg" style={{ background: '#181A20', border: '1px solid #2B3139' }}>
          <div className="flex items-end gap-3">
            <div className="flex-1">
              <label className="block text-sm font-bold mb-1" style={{ color: '#EAECEF' }}>名称（可选）</label>
              <input
                type="text" value={newKeyName} onChange={e => setNewKeyName(e.target.value)}
                placeholder="如：我的实盘、MCP Server"
                className="w-full px-3 py-2 rounded text-sm"
                style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
                onKeyDown={e => e.key === 'Enter' && handleCreate()}
              />
            </div>
            <button onClick={handleCreate} disabled={creating}
              className="px-4 py-2 rounded-lg font-bold text-sm flex items-center gap-1 disabled:opacity-50"
              style={{ background: 'var(--brand-yellow)', color: '#0B0E11' }}>
              <Plus className="w-4 h-4" />
              {creating ? '生成中...' : '生成 Key'}
            </button>
          </div>
        </div>

        {/* Key List */}
        {loading ? (
          <div className="text-center py-8" style={{ color: '#848E9C' }}>加载中...</div>
        ) : keys.length === 0 ? (
          <div className="text-center py-12 rounded-lg" style={{ background: '#181A20', border: '1px solid #2B3139' }}>
            <Key className="w-12 h-12 mx-auto mb-3" style={{ color: '#2B3139' }} />
            <p style={{ color: '#848E9C' }}>暂无 API Key，点击上方按钮创建</p>
          </div>
        ) : (
          <div className="space-y-3">
            {keys.map(key => (
              <div key={key.id} className="p-4 rounded-lg flex items-center justify-between" style={{ background: '#181A20', border: '1px solid #2B3139' }}>
                <div>
                  <div className="flex items-center gap-2 mb-1">
                    <span className="font-mono text-sm" style={{ color: '#EAECEF' }}>{key.key_prefix}</span>
                    {key.name && <span className="text-xs px-2 py-0.5 rounded" style={{ background: '#2B3139', color: '#848E9C' }}>{key.name}</span>}
                  </div>
                  <div className="text-xs" style={{ color: '#5E6673' }}>
                    创建于 {new Date(key.created_at).toLocaleDateString()}
                    {key.last_used && <span className="ml-3">最后使用: {new Date(key.last_used).toLocaleDateString()}</span>}
                  </div>
                </div>
                <button onClick={() => handleDelete(key.id)} disabled={deleting === key.id}
                  className="p-2 rounded-lg disabled:opacity-50" style={{ color: '#F6465D' }} title="撤销">
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            ))}
          </div>
        )}

        {/* Usage */}
        <div className="mt-8 p-4 rounded-lg" style={{ background: '#181A20', border: '1px solid #2B3139' }}>
          <h3 className="font-bold mb-2" style={{ color: '#EAECEF' }}>如何使用</h3>
          <div className="space-y-2 text-sm" style={{ color: '#848E9C' }}>
            <p>1. 点击上方按钮生成一个新的 API Key</p>
            <p>2. 复制 Key（仅展示一次）</p>
            <p>3. 在自建 Hermes 服务中配置环境变量：</p>
            <code className="block mt-2 p-3 rounded font-mono text-xs" style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}>
              {`NOFX_GATEWAY_URL=https://your-nofx-host\nNOFX_API_KEY=nfx_sk_your_key_here`}
            </code>
            <p className="mt-2">使用 Go SDK：<code className="font-mono">gateway/client.New(url, apiKey)</code></p>
            <p className="mt-2">详见 <code className="font-mono">docs/gateway-api.md</code></p>
          </div>
        </div>
      </div>
    </Container>
  )
}
