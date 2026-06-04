import { Component, type ReactNode } from 'react'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
}

export class ChartErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false }

  static getDerivedStateFromError(): State {
    return { hasError: true }
  }

  componentDidCatch(err: Error) {
    console.error('TradeEventPriceChart error:', err)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div
          className="flex items-center justify-center text-sm rounded-lg p-6"
          style={{ minHeight: 200, color: '#848E9C', background: '#0B0E11', border: '1px solid #2B3139' }}
        >
          图表加载失败，请刷新页面重试
        </div>
      )
    }
    return this.props.children
  }
}
