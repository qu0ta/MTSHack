import { describe, expect, it } from 'vitest'
import { chatReducer, initialChatState } from './chatReducer'

describe('chatReducer', () => {
  it('добавляет сообщение пользователя и включает ожидание', () => {
    const state = chatReducer(initialChatState, { type: 'user-sent', text: 'Нужен учёт' })

    expect(state.messages.at(-1)).toMatchObject({ role: 'user', text: 'Нужен учёт' })
    expect(state.isTyping).toBe(true)
  })

  it('добавляет ответ и сохраняет метаданные отладки', () => {
    const state = chatReducer(initialChatState, {
      type: 'assistant-replied',
      reply: {
        text: 'Ответ',
        topic: 'metering',
        sources: ['22-askue.md'],
        responseTimeMs: 820,
      },
    })

    expect(state.messages.at(-1)).toMatchObject({ role: 'assistant', text: 'Ответ' })
    expect(state.lastReply?.sources).toEqual(['22-askue.md'])
    expect(state.isTyping).toBe(false)
  })

  it('возвращает диалог в исходное состояние', () => {
    const changed = chatReducer(initialChatState, { type: 'user-sent', text: 'Вопрос' })

    expect(chatReducer(changed, { type: 'reset' })).toEqual(initialChatState)
  })
})
