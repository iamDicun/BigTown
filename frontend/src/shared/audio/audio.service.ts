import { readonly, ref } from 'vue'

const MUSIC_VOLUME_KEY = 'bigtown:music-volume'
const SFX_VOLUME_KEY = 'bigtown:sfx-volume'
const MUSIC_MUTED_KEY = 'bigtown:music-muted'

const CLICK_SFX_SRC = '/assets/sounds/click.mp3'

type MusicOptions = {
  volume?: number
  fadeMs?: number
  loop?: boolean
}

type PendingMusic = {
  src: string
  options: MusicOptions
}

const musicVolume = ref(readNumber(MUSIC_VOLUME_KEY, 0.1))
const sfxVolume = ref(readNumber(SFX_VOLUME_KEY, 0.15))
const musicMuted = ref(readBool(MUSIC_MUTED_KEY, false))
const audioUnlocked = ref(false)

let currentMusic: HTMLAudioElement | null = null
let currentMusicSrc = ''
let fadeFrame = 0
let pendingMusic: PendingMusic | null = null
let buttonSfxInitialized = false

export const audioState = {
  musicVolume: readonly(musicVolume),
  sfxVolume: readonly(sfxVolume),
  musicMuted: readonly(musicMuted),
  audioUnlocked: readonly(audioUnlocked),
}

export function playMusic(src: string, options: MusicOptions = {}): void {
  pendingMusic = { src, options }

  if (currentMusic && currentMusicSrc === src) {
    applyMusicVolume(currentMusic, options.volume)
    if (currentMusic.paused && !musicMuted.value) void currentMusic.play().catch(() => undefined)
    return
  }

  stopMusic()

  const audio = new Audio(src)
  audio.loop = options.loop ?? true
  audio.volume = 0
  currentMusic = audio
  currentMusicSrc = src

  if (musicMuted.value) return

  void audio
    .play()
    .then(() => {
      audioUnlocked.value = true
      pendingMusic = null
      fadeMusicTo(targetMusicVolume(options.volume), options.fadeMs ?? 1800)
    })
    .catch(() => {
      bindUnlockListener()
    })
}

export function stopMusic(): void {
  pendingMusic = null
  cancelAnimationFrame(fadeFrame)
  fadeFrame = 0
  if (currentMusic) {
    currentMusic.pause()
    currentMusic.currentTime = 0
  }
  currentMusic = null
  currentMusicSrc = ''
}

export function setMusicVolume(value: number): void {
  musicVolume.value = clamp01(value)
  localStorage.setItem(MUSIC_VOLUME_KEY, String(musicVolume.value))
  if (currentMusic) applyMusicVolume(currentMusic)
}

export function setSfxVolume(value: number): void {
  sfxVolume.value = clamp01(value)
  localStorage.setItem(SFX_VOLUME_KEY, String(sfxVolume.value))
}

export function setMusicMuted(value: boolean): void {
  musicMuted.value = value
  localStorage.setItem(MUSIC_MUTED_KEY, String(value))

  if (!currentMusic) return
  if (value) {
    currentMusic.pause()
    return
  }

  currentMusic.volume = 0
  void currentMusic
    .play()
    .then(() => fadeMusicTo(targetMusicVolume(), 900))
    .catch(() => bindUnlockListener())
}

export function toggleMusicMuted(): void {
  setMusicMuted(!musicMuted.value)
}

export function playSfx(src = CLICK_SFX_SRC, volume = 1): void {
  const audio = new Audio(src)
  audio.volume = clamp01(sfxVolume.value * volume)
  void audio.play().catch(() => undefined)
}

export function playRandomSfx(sources: string[], volume = 1): void {
  if (sources.length === 0) return
  const index = Math.floor(Math.random() * sources.length)
  playSfx(sources[index], volume)
}

export function initButtonSfx(): void {
  if (buttonSfxInitialized) return
  buttonSfxInitialized = true

  document.addEventListener('click', (event) => {
    const target = event.target
    if (!(target instanceof Element)) return
    if (!target.closest('button, a, input[type="submit"]')) return
    if (target.closest('input[type="range"]')) return
    playSfx()
  })
}

function applyMusicVolume(audio: HTMLAudioElement, multiplier = 1): void {
  audio.volume = musicMuted.value ? 0 : targetMusicVolume(multiplier)
}

function targetMusicVolume(multiplier = 1): number {
  return clamp01(musicVolume.value * multiplier)
}

function fadeMusicTo(targetVolume: number, durationMs: number): void {
  if (!currentMusic) return

  cancelAnimationFrame(fadeFrame)
  const audio = currentMusic
  const startVolume = audio.volume
  const startedAt = performance.now()

  const tick = (now: number) => {
    const progress = durationMs <= 0 ? 1 : Math.min((now - startedAt) / durationMs, 1)
    audio.volume = clamp01(startVolume + (targetVolume - startVolume) * progress)
    if (progress < 1 && audio === currentMusic) {
      fadeFrame = requestAnimationFrame(tick)
    }
  }

  fadeFrame = requestAnimationFrame(tick)
}

function bindUnlockListener(): void {
  const unlock = () => {
    audioUnlocked.value = true
    if (pendingMusic) {
      const next = pendingMusic
      pendingMusic = null
      playMusic(next.src, next.options)
    }
    window.removeEventListener('pointerdown', unlock)
    window.removeEventListener('keydown', unlock)
  }

  window.addEventListener('pointerdown', unlock, { once: true })
  window.addEventListener('keydown', unlock, { once: true })
}

function readNumber(key: string, fallback: number): number {
  const stored = localStorage.getItem(key)
  if (stored === null) return fallback

  const value = Number(stored)
  return Number.isFinite(value) ? clamp01(value) : fallback
}

function readBool(key: string, fallback: boolean): boolean {
  const value = localStorage.getItem(key)
  if (value === null) return fallback
  return value === 'true'
}

function clamp01(value: number): number {
  return Math.min(Math.max(value, 0), 1)
}
