import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useGameStore } from './game.store'
import * as characterService from '../services/character.service'

vi.mock('../services/character.service')

describe('Game Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('should initialize with default state', () => {
    const store = useGameStore()
    expect(store.characterId).toBeNull()
    expect(store.coins).toBe(0)
    expect(store.mapCode).toBe('village_adventure')
    expect(store.textureKey).toBe('player')
  })

  it('setMyCharacter updates character state and coins', () => {
    const store = useGameStore()
    store.setMyCharacter({
      id: 'char-100',
      user_id: 'user-1',
      name: 'Knight Hero',
      base_asset_key: 'knight',
      coins: 250,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    })

    expect(store.characterId).toBe('char-100')
    expect(store.characterName).toBe('Knight Hero')
    expect(store.characterBaseAssetKey).toBe('knight')
    expect(store.coins).toBe(250)
    expect(store.textureKey).toBe('knight')
  })

  it('loadMyCharacter calls character service and updates store', async () => {
    vi.mocked(characterService.getMe).mockResolvedValueOnce({
      id: 'char-200',
      user_id: 'user-2',
      name: 'Wizard',
      base_asset_key: 'wizard',
      coins: 999,
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    })
    vi.mocked(characterService.getOptions).mockResolvedValueOnce([])

    const store = useGameStore()
    const result = await store.loadMyCharacter()

    expect(result.name).toBe('Wizard')
    expect(store.characterId).toBe('char-200')
    expect(store.coins).toBe(999)
  })

  it('normalizes textureKey when baseAssetKey is cute_fantasy/player_base', () => {
    const store = useGameStore()
    store.characterBaseAssetKey = 'cute_fantasy/player_base'
    expect(store.textureKey).toBe('player')
  })
})
