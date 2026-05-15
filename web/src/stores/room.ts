import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { MemberInfo, MediaState, PlayerState, ChatMessage } from '../types/message'

export const useRoomStore = defineStore('room', () => {
  const roomId = ref('')
  const selfId = ref('')
  const selfToken = ref('')
  const members = ref<MemberInfo[]>([])
  const media = ref<MediaState | null>(null)
  const player = ref<PlayerState>({
    playing: false,
    position: 0,
    updated_at: 0,
  })
  const chatHistory = ref<ChatMessage[]>([])
  const connected = ref(false)

  function setRoomState(state: {
    room_id: string
    self_id: string
    self_token: string
    members: MemberInfo[]
    media: MediaState | null
    player: PlayerState
    chat_history: ChatMessage[]
  }) {
    roomId.value = state.room_id
    selfId.value = state.self_id
    selfToken.value = state.self_token
    members.value = state.members
    media.value = state.media
    player.value = state.player
    chatHistory.value = state.chat_history

    if (state.self_token) {
      sessionStorage.setItem('member_token', state.self_token)
    }
  }

  function addMember(member: MemberInfo) {
    const exists = members.value.find(m => m.id === member.id)
    if (!exists) {
      members.value.push(member)
    }
  }

  function removeMember(id: string) {
    members.value = members.value.filter(m => m.id !== id)
  }

  function addChatMessage(msg: ChatMessage) {
    chatHistory.value.push(msg)
    if (chatHistory.value.length > 200) {
      chatHistory.value = chatHistory.value.slice(-200)
    }
  }

  return {
    roomId,
    selfId,
    selfToken,
    members,
    media,
    player,
    chatHistory,
    connected,
    setRoomState,
    addMember,
    removeMember,
    addChatMessage,
  }
})
