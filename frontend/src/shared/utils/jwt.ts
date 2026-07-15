export interface JwtPayload {
  user_id: string
  role: string
  exp?: number
  [key: string]: unknown
}

// Chỉ decode để đọc role/user_id cho UI (route guard, hiển thị). KHÔNG dùng để xác thực chữ ký
// — verify JWT vẫn hoàn toàn nằm ở backend middleware.
export function decodeJwtPayload(token: string): JwtPayload | null {
  try {
    const segment = token.split('.')[1]
    if (!segment) return null

    const base64 = segment.replace(/-/g, '+').replace(/_/g, '/')
    const json = atob(base64)
    return JSON.parse(json) as JwtPayload
  } catch {
    return null
  }
}
