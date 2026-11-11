import { getAssistantReply, type AssistantReply, type ChatHistoryItem } from '../model/assistantEngine'

export interface AssistantService {
  reply(message: string, history: ChatHistoryItem[]): Promise<AssistantReply>
}

const delay = (duration: number) => new Promise((resolve) => window.setTimeout(resolve, duration))

export const localAssistantService: AssistantService = {
  async reply(message, history) {
    await delay(320)
    return getAssistantReply(message, history)
  },
}
