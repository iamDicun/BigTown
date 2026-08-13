import { test, expect } from '@playwright/test'
import { LoginPage } from '../pages/LoginPage'
import testCases from '../test-data/auth.json'

test.describe('E2E-01 Auth Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.context().clearCookies()
  })

  for (const tc of testCases) {
    test(`${tc.id} - ${tc.description}`, async ({ page }) => {
      const loginPage = new LoginPage(page)

      if (tc.id === 'TC_E2E_AUTH_01') {
        await loginPage.gotoRegister()
        const uniqueEmail = `${tc.input.emailPrefix}${Date.now()}${tc.input.emailDomain}`
        await loginPage.register(tc.input.fullName!, uniqueEmail, tc.input.password)
        await expect(page).toHaveURL(new RegExp(tc.expected.redirectPath), { timeout: 15000 })
      } else if (tc.id === 'TC_E2E_AUTH_02') {
        // Tạo user mới để dùng làm tài khoản "đã có"
        await loginPage.gotoRegister()
        const uniqueEmail = `user_log_${Date.now()}@test.com`
        await loginPage.register('Existing User', uniqueEmail, tc.input.password)
        await page.waitForURL(/\/login/, { timeout: 15000 })

        // Đăng nhập lại bằng chính tài khoản vừa tạo
        await loginPage.gotoLogin()
        await loginPage.login(uniqueEmail, tc.input.password)
        await page.waitForURL(new RegExp(tc.expected.redirectPath))
      }
    })
  }
})
