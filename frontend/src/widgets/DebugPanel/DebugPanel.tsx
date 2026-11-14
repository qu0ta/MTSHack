import type { Message } from '../../features/chat/api/assistantService'
import styles from './DebugPanel.module.css'

export function DebugPanel({ messages }: { messages: Message[] }) {
  const replies = messages.filter((message) => message.role === 'assistant')
  return (
    <div className={styles.panel}>
      {!replies.length && <p>Источники появятся после первого ответа.</p>}
      {replies.map((message) => (
        <div className={styles.sources} key={message.id}>
          <span>AI-{String(messages.findIndex((item) => item.id === message.id) + 1).padStart(2, '0')} / {message.response_time_ms == null ? 'ВРЕМЯ НЕ УКАЗАНО' : `${message.response_time_ms} МС`}</span>
          {message.metadata?.sources?.length ? (
            <ul>{message.metadata.sources.map((source, i) => <li key={i}><b>{String(i + 1).padStart(2, '0')}</b>{source.document}</li>)}</ul>
          ) : <p>Источники не указаны.</p>}
        </div>
      ))}
    </div>
  )
}
