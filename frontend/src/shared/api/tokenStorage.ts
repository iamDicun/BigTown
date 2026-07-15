// Access token chỉ sống trong biến runtime (mất khi reload trang), không lưu localStorage
// để giảm rủi ro XSS. Refresh token nằm trong HttpOnly cookie do backend set, JS không đọc được
// và không cần biết — xem mục 14.1 frontend-architecture-it-asset-tracking.md.
let accessToken: string | null = null

export function getAccessToken(): string | null {
  return accessToken
}

export function setAccessToken(token: string | null): void {
  accessToken = token
}

export function clearAccessToken(): void {
  accessToken = null
}
