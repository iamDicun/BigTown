import { http } from '@/shared/api/http'

// Khớp backend/internal/module/chat/delivery/dto.go ChatMessageResponse.
export interface ChatMessageDto {
  id: string
  room_id: string
  character_id: string
  character_name: string
  message: string
  message_type: string
  created_at: string
}

export function getMessages(roomId: string, limit = 50) {
  return http.get<ChatMessageDto[]>(`/rooms/${roomId}/chat/messages?limit=${limit}`)
}

export function sendMessage(roomId: string, message: string) {
  return http.post<ChatMessageDto>(`/rooms/${roomId}/chat/messages`, { message })
}
