import { readFileSync, existsSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))

function cardsFilePath(): string {
  const root =
    process.env.NOFX_PROJECT_ROOT ||
    join(__dirname, '..', '..', '..')
  return join(root, '.cursor', 'skills', 'hermes', 'exchange-cards.md')
}

export function getExchangeCardSection(exchangeId: string): string {
  const file = cardsFilePath()
  if (!existsSync(file)) {
    return `未找到交易所卡片文件: ${file}`
  }
  const content = readFileSync(file, 'utf8')
  const id = exchangeId.toLowerCase()
  const heading = new RegExp(`^##\\s+${id}\\b[^\\n]*`, 'im')
  const match = content.match(heading)
  if (!match || match.index === undefined) {
    return `未找到交易所 ${exchangeId} 的配置卡片，支持: binance, okx, hyperliquid, aster`
  }
  const start = match.index
  const rest = content.slice(start + match[0].length)
  const nextHeading = rest.search(/\n##\s+/)
  const section = nextHeading >= 0 ? rest.slice(0, nextHeading) : rest
  return `## ${match[0].replace(/^##\s+/, '')}${section}`.trim()
}
