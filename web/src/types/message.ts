export interface Message {
  type: string
  client_msg_id?: string
  payload?: Record<string, unknown>
}

export interface RoomStatePayload {
  room_id: string
  self_id: string
  self_token: string
  members: MemberInfo[]
  media: MediaState | null
  player: PlayerState
  chat_history: ChatMessage[]
}

export interface MemberInfo {
  id: string
  nickname: string
}

export interface MediaState {
  kind: string
  source_url: string
  media_type: string
  title: string
  status: string
  uploader_id?: string
}

export interface PlayerState {
  playing: boolean
  position: number
  updated_at: number
  reason?: string
}

export interface ChatMessage {
  sender_id: string
  nickname: string
  text: string
  ts: number
}

export interface ErrorPayload {
  code: string
  message: string
  cause_msg_id?: string
}

export interface PlayerUpdatePayload {
  playing: boolean
  position: number
  updated_at: number
  reason?: string
}

export interface MediaUpdatePayload {
  kind: string
  source_url: string
  media_type: string
  title: string
  status: string
  uploader_id?: string
}
