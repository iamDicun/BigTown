import { test, expect } from '@playwright/test'
import { LoginPage } from '../pages/LoginPage'
import { GamePage } from '../pages/GamePage'
import testCases from '../test-data/character.json'

test.describe('E2E-02 Character Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.context().clearCookies()
  })

  for (const tc of testCases) {
    test(`${tc.id} - ${tc.description}`, async ({ page }) => {
      const loginPage = new LoginPage(page)
      const gamePage = new GamePage(page)

      // 1. Register new user
      await loginPage.gotoRegister()
      const email = `char_user_${Date.now()}@test.com`
      const password = 'password123'
      await loginPage.register('Char Owner', email, password)
      await page.waitForURL(/\/login/, { timeout: 15000 })

      // 2. Login (app yêu cầu đăng nhập sau khi đăng ký)
      await loginPage.gotoLogin()
      await loginPage.login(email, password)
      await page.waitForURL(/\/character\/create/)

      // 3. Create character
      await gamePage.createCharacter(`${tc.input.name}_${Date.now() % 1000}`)
      await expect(page).toHaveURL(new RegExp(tc.expected.gameUrl))
    })
  }
})
