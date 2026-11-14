export type Chat = { id: string; status: 'active' | 'closed' }
export type Source = { document: string; chunk?: string; score?: number }
export type Message = { id: string; chat_id: string; role: 'user' | 'assistant'; content: string; created_at: string; response_time_ms: number | null; metadata: { sources?: Source[] } | null }
export type ClientData = { contact_name: string; phone: string; company_name: string }
export type Application = { id: string; full_name: string; phone: string; company: string; summary: string; status: string }
export type SendResult = { user_message: Message; assistant_message: Message; needs_manager: boolean; application?: Application | null }

export class ApiError extends Error {
  constructor(public status: number, message: string) { super(message) }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(path, { ...options, signal: options.signal ?? AbortSignal.timeout(120000) })
  if (!response.ok) throw new ApiError(response.status, (await response.text()).trim())
  return response.json() as Promise<T>
}

export const api = {
  createChat: () => request<Chat>('/api/v1/chats', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' }),
  getChat: (id: string) => request<Chat>(`/api/v1/chats/${encodeURIComponent(id)}`),
  async getMessages(id: string) {
    const messages: Message[] = []
    let total: number
    do {
      const page = await request<{ messages: Message[]; total: number }>(`/api/v1/chats/${encodeURIComponent(id)}/messages?limit=100&offset=${messages.length}`)
      messages.push(...page.messages)
      total = page.total
      if (!page.messages.length) break
    } while (messages.length < total)
    return messages
  },
  getApplication: (id: string) => request<Application>(`/api/v1/chats/${encodeURIComponent(id)}/application`),
  send: (id: string, content: string, clientData?: ClientData) => request<SendResult>(`/api/v1/chats/${encodeURIComponent(id)}/messages`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content, ...(clientData ? { client_data: clientData, lead_requested: true } : {}) }),
  }),
}

export function errorMessage(error: unknown): string {
  if (!(error instanceof ApiError)) return 'Нет связи с сервером или время ожидания истекло. Проверьте соединение и обновите историю перед повтором.'
  if (error.status === 404) return 'Чат не найден. Начните новый диалог.'
  if (error.status === 409) return 'Чат закрыт или занят другим запросом. Обновите историю.'
  if (error.status === 422 || error.status === 400 || error.status === 413) return 'Проверьте введённые данные и попробуйте снова.'
  if (error.status === 503) return 'Сервис временно недоступен или занят. Попробуйте позже.'
  if (error.status === 504) return 'Ответ занял слишком много времени. Обновите историю перед повтором.'
  return 'Сервер не смог обработать запрос. Проверьте историю перед повтором.'
}
