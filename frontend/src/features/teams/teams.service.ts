import { app, authentication } from '@microsoft/teams-js'

let initialized = false
let inTeams = false

export async function initTeams(): Promise<boolean> {
  if (initialized) return inTeams
  initialized = true
  try {
    await app.initialize()
    inTeams = true
    return true
  } catch {
    inTeams = false
    return false
  }
}

export function isRunningInTeams(): boolean {
  return inTeams
}

export async function getTeamsSSOToken(): Promise<string> {
  const token = await authentication.getAuthToken()
  return token
}
