import { defineStore } from 'pinia'

type LeaderboardEntry = {
  characterId: string
  name: string
  score: number
}

export const useGameStore = defineStore('game', {
  state: () => ({
    leaderboard: [] as LeaderboardEntry[],
  }),
})
