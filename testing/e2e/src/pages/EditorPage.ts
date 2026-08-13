import { type Page, type Locator } from '@playwright/test'

export class EditorPage {
  readonly page: Page
  readonly coinsBadge: Locator

  constructor(page: Page) {
    this.page = page
    this.coinsBadge = page.getByLabel('Player Coins')
  }
}
