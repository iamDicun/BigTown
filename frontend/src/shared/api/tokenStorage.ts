// Access token chỉ sống trong biến runtime (mất khi reload trang), không lưu localStorage
// để giảm rủi ro XSS. Refresh token nằm trong HttpOnly cookie do backend set, JS không đọc được
// và không cần biết.
// Sử dụng window để tránh lỗi nhân bản module (module duplication) khi Vite chia chunk chạy production.

export function getAccessToken(): string | null {
  if (typeof window !== 'undefined') {
    return (window as any).__accessToken || null
  }
  return null
}

export function setAccessToken(token: string | null): void {
  if (typeof window !== 'undefined') {
    (window as any).__accessToken = token
  }
}

export function clearAccessToken(): void {
  if (typeof window !== 'undefined') {
    (window as any).__accessToken = null
  }
}
