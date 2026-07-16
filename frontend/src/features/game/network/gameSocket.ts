import { Centrifuge, type Subscription } from 'centrifuge'

import { getAccessToken } from '@/shared/api/tokenStorage'

import type { GameClientEvent } from './gameEvents'

type GameSocketOptions = {
  channel?: string
  onEvent?: (event: unknown) => void
}

export const defaultGameChannel = 'room:starter-town'

export function getDefaultRealtimeUrl() {
  const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api'
  const baseUrl = new URL(apiBaseUrl)
  baseUrl.pathname = '/connection/websocket'
  baseUrl.search = ''
  baseUrl.protocol = baseUrl.protocol === 'https:' ? 'wss:' : 'ws:'

  return baseUrl.toString()
}

export function createGameSocket(url: string, options: GameSocketOptions = {}) {
  const token = getAccessToken()
  if (!token) {
    throw new Error('Missing access token for realtime connection')
  }

  const channel = options.channel ?? defaultGameChannel
  const centrifuge = new Centrifuge(url, { token })
  const subscription: Subscription = centrifuge.newSubscription(channel)

  if (options.onEvent) {
    subscription.on('publication', (ctx) => {
      options.onEvent?.(ctx.data)
    })
  }

  subscription.subscribe()
  centrifuge.connect()

  return {
    centrifuge,
    subscription,
    send(event: GameClientEvent) {
      return subscription.publish(event)
    },
    close() {
      subscription.unsubscribe()
      centrifuge.disconnect()
    },
  }
}
