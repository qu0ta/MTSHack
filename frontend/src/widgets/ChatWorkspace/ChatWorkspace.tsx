import { FormEvent, useReducer, useRef, useState } from 'react'
import { localAssistantService } from '../../features/chat/api/assistantService'
import { chatReducer, initialChatState } from '../../features/chat/model/chatReducer'
import { buildLeadSummary } from '../../features/lead/model/lead'
import { DebugPanel } from '../DebugPanel/DebugPanel'
import { LeadForm } from '../LeadForm/LeadForm'
import styles from './ChatWorkspace.module.css'

const welcomeMessage = {
  id: 'welcome',
  role: 'assistant' as const,
  text: 'Здравствуйте. Расскажите, что хотите изменить: снизить расходы, автоматизировать учёт, проверить потери или выйти на оптовый рынок?',
  createdAt: 0,
}

const prompts = [
  'Снизить расходы на электроэнергию',
  'Автоматизировать сбор показаний',
  'Оценить солнечную станцию',
]

export function ChatWorkspace() {
  const [state, dispatch] = useReducer(chatReducer, { ...initialChatState, messages: [welcomeMessage] })
  const [input, setInput] = useState('')
  const [isDebugOpen, setIsDebugOpen] = useState(false)
  const [isLeadOpen, setIsLeadOpen] = useState(false)
  const [isComplete, setIsComplete] = useState(false)
  const feedRef = useRef<HTMLDivElement>(null)

  const send = async (text: string) => {
    const cleanText = text.trim()
    if (!cleanText || state.isTyping) return
    dispatch({ type: 'user-sent', text: cleanText })
    setInput('')

    const reply = await localAssistantService.reply(cleanText, state.messages)
    dispatch({ type: 'assistant-replied', reply })
    window.setTimeout(() => {
      const feed = feedRef.current
      if (feed && typeof feed.scrollTo === 'function') {
        feed.scrollTo({ top: feed.scrollHeight, behavior: 'smooth' })
      }
    }, 0)
  }

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    void send(input)
  }

  const reset = () => {
    dispatch({ type: 'reset' })
    setIsDebugOpen(false)
    setIsLeadOpen(false)
    setIsComplete(false)
  }

  return (
    <div className={styles.workspace}>
      <div className={styles.chat}>
        <div className={styles.chatTop}>
          <div><i /> КОНСУЛЬТАНТ НА СВЯЗИ</div>
          <button type="button" onClick={reset}>Очистить</button>
        </div>

        <div className={styles.feed} ref={feedRef} aria-live="polite">
          {state.messages.map((message, index) => (
            <article className={`${styles.message} ${styles[message.role]}`} key={message.id}>
              <span>{message.role === 'assistant' ? `AI-${String(index + 1).padStart(2, '0')}` : 'ВЫ'}</span>
              <p>{message.text}</p>
            </article>
          ))}
          {state.isTyping && (
            <div className={styles.typing} aria-label="Консультант формирует ответ"><i /><i /><i /></div>
          )}
        </div>

        {state.messages.length === 1 && (
          <div className={styles.prompts}>
            {prompts.map((prompt) => <button type="button" key={prompt} onClick={() => void send(prompt)}>{prompt}<span aria-hidden="true">→</span></button>)}
          </div>
        )}

        <form className={styles.composer} onSubmit={handleSubmit}>
          <label htmlFor="assistant-message">Сообщение консультанту</label>
          <textarea
            id="assistant-message"
            value={input}
            onChange={(event) => setInput(event.target.value)}
            placeholder="Например: у нас три склада и 40 счётчиков…"
            rows={2}
            onKeyDown={(event) => {
              if (event.key === 'Enter' && !event.shiftKey) {
                event.preventDefault()
                event.currentTarget.form?.requestSubmit()
              }
            }}
          />
          <button type="submit" disabled={!input.trim() || state.isTyping} aria-label="Отправить сообщение">
            <svg aria-hidden="true" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round">
              <path d="M12 19V5M6 11l6-6 6 6" />
            </svg>
          </button>
        </form>
      </div>

      <aside className={styles.rail}>
        <button
          className={styles.debugButton}
          type="button"
          onClick={() => setIsDebugOpen((value) => !value)}
          aria-expanded={isDebugOpen}
          aria-label={`${isDebugOpen ? 'Закрыть' : 'Открыть'} режим отладки`}
        >
          <span><i /> Режим отладки</span>
          <b>{isDebugOpen ? '−' : '+'}</b>
        </button>

        {isDebugOpen && <DebugPanel reply={state.lastReply} />}

        <div className={styles.handoff}>
          <span>НУЖЕН РАСЧЁТ?</span>
          <h2>Передать эксперту</h2>
          <p>Контакт и суть разговора уйдут менеджеру. Повторять всё заново не придётся.</p>
          <button type="button" onClick={() => setIsLeadOpen(true)}>Передать менеджеру <b>↗</b></button>
        </div>

        {isLeadOpen && (
          <LeadForm
            summary={buildLeadSummary(state.messages)}
            isComplete={isComplete}
            onComplete={() => setIsComplete(true)}
            onClose={() => setIsLeadOpen(false)}
          />
        )}
      </aside>
    </div>
  )
}
