export function formatDuration(duration: string | undefined): string {
  if (!duration) return '-'

  const match = duration.match(/(\d+h)?(\d+m)?(\d+\.?\d*s)?/)
  if (!match) return duration

  const hours = match[1] || ''
  const minutes = match[2] || ''
  const seconds = match[3] || ''

  let result = ''
  if (hours) result += hours.replace('h', '小时')
  if (minutes) result += minutes.replace('m', '分')
  if (!hours && seconds) result += seconds.replace(/(\d+)\.?\d*s/, '$1秒')

  return result || duration
}
