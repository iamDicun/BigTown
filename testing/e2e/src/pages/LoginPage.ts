import { type Page, type Locator } from '@playwright/test'

export class LoginPage {
  readonly page: Page
  readonly emailInput: Locator
  readonly passwordInput: Locator
  readonly loginButton: Locator
  readonly registerTabLink: Locator
  readonly fullNameInput: Locator
  readonly registerButton: Locator

  constructor(page: Page) {
    this.page = page
    this.emailInput = page.locator('input[autocomplete="email"]')
    this.passwordInput = page.locator('input[autocomplete="current-password"], input[autocomplete="new-password"]')
    this.loginButton = page.getByRole('button', { name: 'Đăng nhập' })
    this.registerTabLink = page.getByRole('link', { name: 'Đăng ký' })
    this.fullNameInput = page.locator('input[autocomplete="name"]')
    this.registerButton = page.getByRole('button', { name: 'Đăng ký' })
  }

  async gotoLogin() {
    await this.page.goto('/login')
    await this.emailInput.waitFor({ state: 'visible', timeout: 10000 })
  }

  async gotoRegister() {
    await this.page.goto('/login')
    await this.registerTabLink.click()
    await this.fullNameInput.waitFor({ state: 'visible', timeout: 10000 })
  }

  async login(email: string, pass: string) {
    await this.emailInput.fill(email)
    await this.passwordInput.fill(pass)
    await this.loginButton.click()
  }

  async register(fullName: string, email: string, pass: string) {
    await this.fullNameInput.fill(fullName)
    await this.emailInput.fill(email)
    await this.passwordInput.fill(pass)
    await this.registerButton.click()
  }
}
