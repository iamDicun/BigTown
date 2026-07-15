import type { GameClientEvent } from './gameEvents'

export function createGameSocket(url: string) {
  const socket = new WebSocket(url)

  return {
    socket,
    send(event: GameClientEvent) {
      if (socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify(event))
      }
    },
    close() {
      socket.close()
    },
  }
}
