import { test, expect } from '@playwright/test'
import { LoginPage } from '../pages/LoginPage'
import { GamePage } from '../pages/GamePage'
import testCases from '../test-data/chat.json'

test.describe('E2E-04 Chat Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.context().clearCookies()
  })

  for (const tc of testCases) {
    test(`${tc.id} - ${tc.description}`, async ({ page }) => {
      const loginPage = new LoginPage(page)
      const gamePage = new GamePage(page)

      // 1. Register & Enter game
      await loginPage.gotoRegister()
      const email = `chat_user_${Date.now()}@test.com`
      const password = 'password123'
      await loginPage.register('Chatter', email, password)
      await page.waitForURL(/\/login/, { timeout: 15000 })

      // 2. Login (app yêu cầu đăng nhập sau khi đăng ký)
      await loginPage.gotoLogin()
      await loginPage.login(email, password)
      await page.waitForURL(/\/character\/create/)
      await gamePage.createCharacter('ChatHero')
      await page.waitForURL(/\/game/)

      // 3. Send Chat message
      await gamePage.sendChatMessage(tc.input.message)

      // 4. Verify message is visible in chat history
      await expect(gamePage.chatMessagesContainer).toContainText(tc.input.message, { timeout: 15000 })
    })
  }
})
