import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  ArrowDown,
  ArrowUp,
  GripVertical,
  RefreshCw,
  RotateCcw,
  Save,
  Star,
} from 'lucide-react'
import { useSymbolPreferences } from '../contexts/SymbolPreferencesContext'
import { sortSymbolsByPreference } from '../lib/sortSymbols'
import { normalizeTradingSymbol } from '../components/trade-events/tradeEventChartUtils'

export function SymbolManagementPage() {
  const {
    preferences,
    notionalBySymbol,
    loading,
    saveOrder,
    saveStarred,
    resetToDefault,
    refresh,
  } = useSymbolPreferences()

  const [items, setItems] = useState<string[]>([])
  const [starred, setStarred] = useState<string[]>([])
  const [dragIndex, setDragIndex] = useState<number | null>(null)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  const catalog = preferences?.catalog ?? []

  const sortOptions = useMemo(
    () => ({
      customOrder: preferences?.symbols,
      useCustomOrder: preferences?.useCustomOrder,
      notionalBySymbol,
      starredOrder: starred,
    }),
    [preferences, notionalBySymbol, starred]
  )

  const syncItemsFromPreferences = useCallback(() => {
    if (!preferences) return
    setStarred(preferences.starredSymbols ?? [])
    const base =
      preferences.useCustomOrder && preferences.symbols.length > 0
        ? preferences.symbols
        : sortSymbolsByPreference(catalog, {
            notionalBySymbol,
            starredOrder: preferences.starredSymbols,
          })
    const merged = [...base]
    for (const sym of catalog) {
      if (!merged.includes(sym)) merged.push(sym)
    }
    setItems(
      sortSymbolsByPreference(merged, {
        customOrder: preferences.useCustomOrder ? preferences.symbols : undefined,
        useCustomOrder: preferences.useCustomOrder,
        notionalBySymbol,
        starredOrder: preferences.starredSymbols,
      })
    )
  }, [preferences, catalog, notionalBySymbol])

  useEffect(() => {
    if (!loading && preferences) {
      syncItemsFromPreferences()
    }
  }, [loading, preferences, syncItemsFromPreferences])

  const starredSet = useMemo(() => new Set(starred), [starred])

  const addableSymbols = useMemo(
    () =>
      sortSymbolsByPreference(
        catalog.filter((s) => !items.includes(s)),
        sortOptions
      ),
    [catalog, items, sortOptions]
  )

  const moveItem = (from: number, to: number) => {
    if (to < 0 || to >= items.length || from === to) return
    setItems((prev) => {
      const next = [...prev]
      const [picked] = next.splice(from, 1)
      next.splice(to, 0, picked)
      return next
    })
  }

  const handleDragStart = (index: number) => setDragIndex(index)

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault()
    if (dragIndex === null || dragIndex === index) return
    moveItem(dragIndex, index)
    setDragIndex(index)
  }

  const handleDragEnd = () => setDragIndex(null)

  const handleSave = async () => {
    setSaving(true)
    setError('')
    setMessage('')
    try {
      await saveStarred(starred)
      await saveOrder(items, true)
      setMessage('排序与特别关注已保存')
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleReset = async () => {
    setSaving(true)
    setError('')
    setMessage('')
    try {
      await resetToDefault()
      setMessage('已恢复为按持仓价值排序（特别关注仍生效）')
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '恢复失败')
    } finally {
      setSaving(false)
    }
  }

  const handleToggleStar = async (sym: string) => {
    const n = normalizeTradingSymbol(sym)
    if (!n) return
    const next = starredSet.has(n)
      ? starred.filter((s) => s !== n)
      : [...starred, n]
    setStarred(next)
    setItems((prev) =>
      sortSymbolsByPreference(prev, {
        ...sortOptions,
        starredOrder: next,
        customOrder: items,
        useCustomOrder: true,
      })
    )
    setError('')
    try {
      await saveStarred(next)
      setMessage(starredSet.has(n) ? `已取消特别关注 ${n}` : `已添加特别关注 ${n}`)
    } catch (e: unknown) {
      setStarred(starred)
      setError(e instanceof Error ? e.message : '保存特别关注失败')
    }
  }

  const handleAddSymbol = (sym: string) => {
    const n = normalizeTradingSymbol(sym)
    if (!n || items.includes(n)) return
    setItems((prev) =>
      sortSymbolsByPreference([...prev, n], {
        ...sortOptions,
        customOrder: [...items, n],
        useCustomOrder: true,
      })
    )
  }

  const formatNotional = (sym: string) => {
    const v = notionalBySymbol[sym] ?? 0
    if (v <= 0) return '—'
    return `$${v.toLocaleString('zh-CN', { maximumFractionDigits: 2 })}`
  }

  return (
    <div className="max-w-3xl mx-auto">
      <div className="mb-6">
        <h1 className="text-2xl font-bold mb-2" style={{ color: '#EAECEF' }}>
          币种管理
        </h1>
        <p className="text-sm" style={{ color: '#848E9C' }}>
          调整币种显示顺序，全站币种下拉与快速选择将按此排序。特别关注币种始终排在最前；未自定义排序时，其余币种按持仓名义价值（mark_price
          × quantity）倒序。
        </p>
      </div>

      <div className="flex flex-wrap gap-2 mb-4">
        <button
          type="button"
          onClick={handleSave}
          disabled={saving || loading}
          className="inline-flex items-center gap-2 px-4 py-2 rounded text-sm font-semibold disabled:opacity-50"
          style={{ background: '#F0B90B', color: '#0B0E11' }}
        >
          <Save className="w-4 h-4" />
          保存排序
        </button>
        <button
          type="button"
          onClick={handleReset}
          disabled={saving || loading}
          className="inline-flex items-center gap-2 px-4 py-2 rounded text-sm font-semibold disabled:opacity-50"
          style={{ background: '#2B3139', color: '#EAECEF' }}
        >
          <RotateCcw className="w-4 h-4" />
          恢复默认
        </button>
        <button
          type="button"
          onClick={() => refresh()}
          disabled={loading}
          className="inline-flex items-center gap-2 px-4 py-2 rounded text-sm disabled:opacity-50"
          style={{ background: '#1E2329', color: '#848E9C', border: '1px solid #2B3139' }}
        >
          <RefreshCw className="w-4 h-4" />
          刷新价值
        </button>
        {addableSymbols.length > 0 && (
          <select
            className="text-sm px-3 py-2 rounded"
            style={{ background: '#0B0E11', color: '#EAECEF', border: '1px solid #2B3139' }}
            defaultValue=""
            onChange={(e) => {
              if (e.target.value) {
                handleAddSymbol(e.target.value)
                e.target.value = ''
              }
            }}
          >
            <option value="">添加币种…</option>
            {addableSymbols.map((s) => (
              <option key={s} value={s}>
                {s.replace(/USDT$/i, '')}
                {starredSet.has(s) ? ' ★' : ''}
              </option>
            ))}
          </select>
        )}
      </div>

      {message && (
        <p className="text-xs mb-3" style={{ color: '#0ECB81' }}>
          {message}
        </p>
      )}
      {error && (
        <p className="text-xs mb-3" style={{ color: '#F6465D' }}>
          {error}
        </p>
      )}

      <div
        className="rounded-lg overflow-hidden"
        style={{ border: '1px solid #2B3139', background: '#0B0E11' }}
      >
        <div
          className="grid grid-cols-[40px_48px_1fr_120px_80px] gap-2 px-4 py-2 text-xs font-semibold"
          style={{ color: '#848E9C', borderBottom: '1px solid #2B3139', background: '#181A20' }}
        >
          <span />
          <span className="text-center">关注</span>
          <span>币种</span>
          <span className="text-right">持仓价值</span>
          <span className="text-center">操作</span>
        </div>

        {loading ? (
          <div className="py-12 text-center text-sm" style={{ color: '#5E6673' }}>
            加载中…
          </div>
        ) : items.length === 0 ? (
          <div className="py-12 text-center text-sm" style={{ color: '#5E6673' }}>
            暂无币种
          </div>
        ) : (
          items.map((sym, index) => {
            const isStarred = starredSet.has(sym)
            return (
              <div
                key={sym}
                draggable
                onDragStart={() => handleDragStart(index)}
                onDragOver={(e) => handleDragOver(e, index)}
                onDragEnd={handleDragEnd}
                className="grid grid-cols-[40px_48px_1fr_120px_80px] gap-2 px-4 py-3 items-center text-sm"
                style={{
                  borderBottom: '1px solid #1E2329',
                  background:
                    dragIndex === index
                      ? 'rgba(240,185,11,0.08)'
                      : isStarred
                        ? 'rgba(240,185,11,0.04)'
                        : 'transparent',
                  cursor: 'grab',
                }}
              >
                <GripVertical className="w-4 h-4" style={{ color: '#5E6673' }} />
                <div className="flex justify-center">
                  <button
                    type="button"
                    onClick={() => handleToggleStar(sym)}
                    className="p-1 rounded transition-colors"
                    aria-label={isStarred ? '取消特别关注' : '添加特别关注'}
                    title={isStarred ? '取消特别关注' : '添加特别关注'}
                  >
                    <Star
                      className="w-4 h-4"
                      style={{
                        color: isStarred ? '#F0B90B' : '#5E6673',
                        fill: isStarred ? '#F0B90B' : 'transparent',
                      }}
                    />
                  </button>
                </div>
                <span className="font-mono font-semibold" style={{ color: '#EAECEF' }}>
                  {sym.replace(/USDT$/i, '')}
                  <span className="text-xs ml-2 font-normal" style={{ color: '#5E6673' }}>
                    {sym}
                  </span>
                  {isStarred && (
                    <span
                      className="text-xs ml-2 font-normal"
                      style={{ color: '#F0B90B' }}
                    >
                      特别关注
                    </span>
                  )}
                </span>
                <span className="text-right font-mono" style={{ color: '#848E9C' }}>
                  {formatNotional(sym)}
                </span>
                <div className="flex justify-center gap-1">
                  <button
                    type="button"
                    disabled={index === 0}
                    onClick={() => moveItem(index, index - 1)}
                    className="p-1 rounded disabled:opacity-30"
                    style={{ color: '#848E9C' }}
                    aria-label="上移"
                  >
                    <ArrowUp className="w-4 h-4" />
                  </button>
                  <button
                    type="button"
                    disabled={index === items.length - 1}
                    onClick={() => moveItem(index, index + 1)}
                    className="p-1 rounded disabled:opacity-30"
                    style={{ color: '#848E9C' }}
                    aria-label="下移"
                  >
                    <ArrowDown className="w-4 h-4" />
                  </button>
                </div>
              </div>
            )
          })
        )}
      </div>

      {preferences && (
        <p className="text-xs mt-3" style={{ color: '#5E6673' }}>
          当前模式：{preferences.useCustomOrder ? '自定义排序' : '按持仓价值自动排序'}
          {starred.length > 0 && ` · 特别关注 ${starred.length} 个`}
        </p>
      )}
    </div>
  )
}
