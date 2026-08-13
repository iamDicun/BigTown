import { test, expect } from '@playwright/test'
import { LoginPage } from '../pages/LoginPage'
import { GamePage } from '../pages/GamePage'
import { EditorPage } from '../pages/EditorPage'
import testCases from '../test-data/editor.json'

test.describe('E2E-03 Editor Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.context().clearCookies()
  })

  for (const tc of testCases) {
    test(`${tc.id} - ${tc.description}`, async ({ page }) => {
      const loginPage = new LoginPage(page)
      const gamePage = new GamePage(page)
      const editorPage = new EditorPage(page)

      // 1. Register & create character
      await loginPage.gotoRegister()
      const email = `editor_user_${Date.now()}@test.com`
      const password = 'password123'
      await loginPage.register('Editor User', email, password)
      await page.waitForURL(/\/login/, { timeout: 15000 })

      // 2. Login (app yêu cầu đăng nhập sau khi đăng ký)
      await loginPage.gotoLogin()
      await loginPage.login(email, password)
      await page.waitForURL(/\/character\/create/)
      await gamePage.createCharacter('BuilderHero')
      await page.waitForURL(/\/game/)

      // 3. Verify Coins badge is visible
      await expect(editorPage.coinsBadge).toBeVisible({ timeout: 15000 })
    })
  }
})
