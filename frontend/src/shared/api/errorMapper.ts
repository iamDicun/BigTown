import { ApiError } from './http'

export function toErrorMessage(err: unknown, fallback = 'Đã có lỗi xảy ra'): string {
  if (err instanceof ApiError) return err.message
  if (err instanceof Error) return err.message
  return fallback
}
