import { useEffect, useRef, useState, type FormEvent } from 'react'
import { api, ApiError, errorMessage, type Application, type Message } from '../../features/chat/api/assistantService'
import type { LeadData } from '../../features/lead/model/lead'
import { DebugPanel } from '../DebugPanel/DebugPanel'
import { LeadForm } from '../LeadForm/LeadForm'
import styles from './ChatWorkspace.module.css'

const storageKey = 'energytool-chat-id'
const prompts = ['Снизить расходы на электроэнергию', 'Автоматизировать сбор показаний', 'Оценить солнечную станцию']

export function ChatWorkspace() {
  const [chatId, setChatId] = useState(() => localStorage.getItem(storageKey))
  const [messages, setMessages] = useState<Message[]>([])
  const [application, setApplication] = useState<Application | null>(null)
  const [input, setInput] = useState('')
  const [pending, setPending] = useState(false)
  const [loading, setLoading] = useState(Boolean(chatId))
  const [closed, setClosed] = useState(false)
  const [error, setError] = useState('')
  const [uncertain, setUncertain] = useState(false)
  const [debugOpen, setDebugOpen] = useState(false)
  const [leadOpen, setLeadOpen] = useState(false)
  const generation = useRef(0)
  const sending = useRef(false)
  const feedRef = useRef<HTMLDivElement>(null)

  async function loadChat(id: string, current: number) {
    const [chat, history] = await Promise.all([api.getChat(id), api.getMessages(id)])
    let saved: Application | null = null
    try { saved = await api.getApplication(id) }
    catch (err) { if (!(err instanceof ApiError && err.status === 404)) throw err }
    if (generation.current !== current) return
    setMessages(history)
    setApplication(saved)
    setClosed(chat.status === 'closed')
    setUncertain(false)
    setError('')
  }

  useEffect(() => {
    if (!chatId || sending.current) return
    const current = generation.current
    let mounted = true
    void loadChat(chatId, current).catch((err) => {
      if (mounted && generation.current === current) { setError(errorMessage(err)); setUncertain(true) }
    }).finally(() => { if (mounted && generation.current === current) setLoading(false) })
    return () => { mounted = false }
  }, [chatId])

  async function refresh() {
    if (!chatId || sending.current) return
    const current = generation.current
    setLoading(true)
    try { await loadChat(chatId, current) }
    catch (err) { if (generation.current === current) { setError(errorMessage(err)); setUncertain(true) } }
    finally { if (generation.current === current) setLoading(false) }
  }

  async function send(text: string, lead?: LeadData) {
    const content = text.trim()
    if (!content || sending.current || loading || closed || uncertain) return
    if (Array.from(content).length > 10000) { setError('Сообщение должно быть не длиннее 10000 символов.'); return }
    const current = generation.current
    sending.current = true
    setPending(true)
    setError('')
    let id = chatId
    let sent = false
    try {
      if (!id) {
        const chat = await api.createChat()
        if (generation.current !== current) return
        id = chat.id
        localStorage.setItem(storageKey, id)
        setChatId(id)
      }
      sent = true
      const result = await api.send(id, content, lead ? { contact_name: lead.name.trim(), phone: lead.phone.trim(), company_name: lead.company.trim() } : undefined)
      if (generation.current !== current) return
      setMessages((old) => [...old.filter((item) => item.id !== result.user_message.id && item.id !== result.assistant_message.id), result.user_message, result.assistant_message])
      if (result.application?.id) setApplication(result.application)
      else if (lead) setError('Заявка пока не сохранена. Проверьте ответ консультанта.')
      if (!lead) setInput('')
      setTimeout(() => feedRef.current?.scrollTo?.({ top: feedRef.current.scrollHeight, behavior: 'smooth' }), 0)
    } catch (err) {
      if (generation.current !== current) return
      setError(errorMessage(err))
      if (!lead) setInput(content)
      if (sent && (!(err instanceof ApiError) || err.status === 504 || err.status === 409 || err.status >= 500)) setUncertain(true)
    } finally {
      if (generation.current === current) { sending.current = false; setPending(false) }
    }
  }

  function newChat() {
    generation.current += 1
    sending.current = false
    localStorage.removeItem(storageKey)
    setChatId(null); setMessages([]); setApplication(null); setInput('')
    setPending(false); setLoading(false); setClosed(false); setError(''); setUncertain(false); setDebugOpen(false); setLeadOpen(false)
  }

  function submit(event: FormEvent) { event.preventDefault(); void send(input) }
  const disabled = pending || loading || closed || uncertain

  return (
    <div className={styles.workspace}>
      <div className={styles.chat}>
        <div className={styles.chatTop}>
          <div><i /> {loading ? 'ЗАГРУЖАЕМ ДИАЛОГ' : pending ? 'ОЖИДАЕМ ОТВЕТ' : closed ? 'ДИАЛОГ ЗАКРЫТ' : 'КОНСУЛЬТАЦИЯ'}</div>
          <button type="button" onClick={newChat}>Новый диалог</button>
        </div>
        <div className={styles.feed} ref={feedRef} aria-live="polite">
          {!messages.length && !loading && <article className={styles.message}><span>AI</span><p>Здравствуйте. Расскажите, что хотите изменить: снизить расходы, автоматизировать учёт, проверить потери или выйти на оптовый рынок?</p></article>}
          {messages.map((message, index) => (
            <article className={`${styles.message} ${styles[message.role]}`} key={message.id}>
              <span>{message.role === 'assistant' ? `AI-${String(index + 1).padStart(2, '0')}` : 'ВЫ'}</span>
              <p>{message.content}</p>
            </article>
          ))}
          {pending && <div className={styles.typing} aria-label="Консультант формирует ответ"><i /><i /><i /></div>}
          {error && <div role="alert" className={styles.error}>{error} {uncertain && chatId && <button type="button" onClick={() => void refresh()} disabled={loading}>Обновить историю</button>}</div>}
        </div>
        {!messages.length && !disabled && <div className={styles.prompts}>{prompts.map((prompt) => <button type="button" key={prompt} onClick={() => { setInput(prompt); void send(prompt) }}>{prompt}<span aria-hidden="true">→</span></button>)}</div>}
        <form className={styles.composer} onSubmit={submit}>
          <label htmlFor="assistant-message">Сообщение консультанту</label>
          <textarea id="assistant-message" value={input} onChange={(event) => setInput(event.target.value)} placeholder="Например: у нас три склада и 40 счётчиков..." rows={2} disabled={pending || closed} onKeyDown={(event) => {
            if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing) { event.preventDefault(); event.currentTarget.form?.requestSubmit() }
          }} />
          <button type="submit" disabled={!input.trim() || disabled} aria-label="Отправить сообщение">
            <svg aria-hidden="true" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round"><path d="M12 19V5M6 11l6-6 6 6" /></svg>
          </button>
        </form>
      </div>
      <aside className={styles.rail}>
        <button className={styles.debugButton} type="button" onClick={() => setDebugOpen((value) => !value)} aria-expanded={debugOpen} aria-label={`${debugOpen ? 'Закрыть' : 'Открыть'} режим отладки`}>
          <span><i /> Режим отладки</span><b>{debugOpen ? '-' : '+'}</b>
        </button>
        {debugOpen && <DebugPanel messages={messages} />}
        <div className={styles.handoff}>
          <span>НУЖЕН РАСЧЁТ?</span><h2>Передать эксперту</h2>
          <p>Оставьте контакт, чтобы сохранить заявку с кратким описанием задачи.</p>
          <button type="button" onClick={() => setLeadOpen(true)} disabled={pending || loading || closed}>Заявка менеджеру <b>↗</b></button>
        </div>
        {leadOpen && <LeadForm application={application} pending={pending || loading} error={error} uncertain={uncertain} onRefresh={() => void refresh()} onClose={() => setLeadOpen(false)} onSubmit={(lead) => void send('Прошу связаться со мной по вопросу консультации.', lead)} />}
      </aside>
    </div>
  )
}
