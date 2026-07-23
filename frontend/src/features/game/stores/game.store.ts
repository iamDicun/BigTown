import { defineStore } from 'pinia'

import { defaultConfig as defaultSpritesheetConfig } from '../phaser/playerAnimations'
import * as characterService from '../services/character.service'
import type { SpritesheetConfigDto } from '../services/character.service'

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
    characterBaseAssetKey: null as string | null,
    spritesheetConfig: defaultSpritesheetConfig() as SpritesheetConfigDto,
  }),
  getters: {
    textureKey(): string {
      return this.characterBaseAssetKey ?? 'player'
    },
  },
  actions: {
    async loadMyCharacter() {
      const character = await characterService.getMe()
      this.characterId = character.id
      this.characterName = character.name
      this.characterBaseAssetKey = character.base_asset_key
      await this.loadMatchingConfig(character.base_asset_key)
      return character
    },

    setMyCharacter(character: characterService.CharacterDto) {
      this.characterId = character.id
      this.characterName = character.name
      this.characterBaseAssetKey = character.base_asset_key
    },

    setSpritesheetConfig(config: SpritesheetConfigDto) {
      this.spritesheetConfig = config
    },

    async loadMatchingConfig(baseAssetKey: string) {
      try {
        const options = await characterService.getOptions()
        const match = options.find((o) => o.base_asset_key === baseAssetKey)
        if (match) {
          this.spritesheetConfig = match.spritesheet
        }
      } catch {
        // Nếu không load được options (mạng lỗi...), giữ config mặc định từ defaultConfig().
      }
    },
  },
})
