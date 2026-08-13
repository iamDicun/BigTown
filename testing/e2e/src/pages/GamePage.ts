import { type Page, type Locator } from '@playwright/test'

export class GamePage {
  readonly page: Page
  readonly characterNameInput: Locator
  readonly createCharacterButton: Locator
  readonly chatInput: Locator
  readonly sendChatButton: Locator
  readonly chatMessagesContainer: Locator

  constructor(page: Page) {
    this.page = page
    this.characterNameInput = page.locator('input[placeholder*="BigCat"]')
    this.createCharacterButton = page.getByRole('button', { name: 'Tạo nhân vật' })
    this.chatInput = page.getByPlaceholder('Nhắn trong map...')
    this.sendChatButton = page.getByRole('button', { name: 'Gửi' })
    this.chatMessagesContainer = page.locator('.messages')
  }

  async createCharacter(name: string) {
    await this.characterNameInput.fill(name)
    await this.createCharacterButton.click()
  }

  async sendChatMessage(msg: string) {
    await this.chatInput.fill(msg)
    await this.sendChatButton.click()
  }
}
