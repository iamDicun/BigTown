import { describe, expect, it, vi } from 'vitest'
import { getDefaultRealtimeUrl } from './gameSocket'

describe('GameSocket Network', () => {
  it('getDefaultRealtimeUrl converts http/https VITE_API_BASE_URL to ws/wss connection URL', () => {
    import.meta.env.VITE_API_BASE_URL = 'http://localhost:8080'
    const urlHttp = getDefaultRealtimeUrl()
    expect(urlHttp).toBe('ws://localhost:8080/connection/websocket')

    import.meta.env.VITE_API_BASE_URL = 'https://game.bigtown.com'
    const urlHttps = getDefaultRealtimeUrl()
    expect(urlHttps).toBe('wss://game.bigtown.com/connection/websocket')
  })

  it('getDefaultRealtimeUrl handles relative URL with window.location.origin', () => {
    vi.stubGlobal('window', {
      location: {
        origin: 'http://localhost:3000',
      },
    })
    import.meta.env.VITE_API_BASE_URL = '/api'
    const url = getDefaultRealtimeUrl()
    expect(url).toBe('ws://localhost:3000/connection/websocket')
    vi.unstubAllGlobals()
  })
})
