import { defineStore } from 'pinia'

import * as characterService from '../services/character.service'

type LeaderboardEntry = {
  characterId: string
  name: string
  score: number
}

export const useGameStore = defineStore('game', {
  state: () => ({
    leaderboard: [] as LeaderboardEntry[],
    characterId: null as string | null,
    characterName: null as string | null,
  }),
  actions: {
    // Gọi 1 lần khi vào GameView để biết character_id của chính mình — dùng để so sánh
    // "mine" cho chat bubble/panel và làm local player id khi nối RoomStore ở phase sau.
    async loadMyCharacter() {
      const character = await characterService.getMe()
      this.characterId = character.id
      this.characterName = character.name
      return character
    },
  },
})
