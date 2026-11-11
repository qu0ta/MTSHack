import type { AssistantReply } from './assistantEngine'

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  text: string
  createdAt: number
}

export interface ChatState {
  messages: ChatMessage[]
  isTyping: boolean
  lastReply: AssistantReply | null
}

export type ChatAction =
  | { type: 'user-sent'; text: string }
  | { type: 'assistant-replied'; reply: AssistantReply }
  | { type: 'reset' }

export const initialChatState: ChatState = {
  messages: [],
  isTyping: false,
  lastReply: null,
}

export function chatReducer(state: ChatState, action: ChatAction): ChatState {
  if (action.type === 'reset') {
    return initialChatState
  }

  if (action.type === 'user-sent') {
    const message: ChatMessage = {
      id: `user-${Date.now()}-${state.messages.length}`,
      role: 'user',
      text: action.text,
      createdAt: Date.now(),
    }

    return {
      ...state,
      messages: [...state.messages, message],
      isTyping: true,
    }
  }

  const message: ChatMessage = {
    id: `assistant-${Date.now()}-${state.messages.length}`,
    role: 'assistant',
    text: action.reply.text,
    createdAt: Date.now(),
  }

  return {
    messages: [...state.messages, message],
    isTyping: false,
    lastReply: action.reply,
  }
}
