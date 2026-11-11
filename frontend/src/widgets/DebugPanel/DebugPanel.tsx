import type { AssistantReply } from '../../features/chat/model/assistantEngine'
import styles from './DebugPanel.module.css'

interface DebugPanelProps {
  reply: AssistantReply | null
}

export function DebugPanel({ reply }: DebugPanelProps) {
  return (
    <div className={styles.panel}>
      <dl>
        <div><dt>STATUS</dt><dd><i /> {reply ? 'READY' : 'WAITING'}</dd></div>
        <div><dt>ВРЕМЯ ОТВЕТА</dt><dd>{reply ? `${reply.responseTimeMs} MS` : '-'}</dd></div>
        <div><dt>TOPIC</dt><dd>{reply?.topic.toUpperCase() ?? '-'}</dd></div>
      </dl>
      <div className={styles.sources}>
        <span>KNOWLEDGE SOURCES</span>
        {reply?.sources.length ? (
          <ul>{reply.sources.map((source, index) => <li key={source}><b>{String(index + 1).padStart(2, '0')}</b>{source}</li>)}</ul>
        ) : <p>Источники появятся после первого ответа.</p>}
      </div>
    </div>
  )
}
