import { ChatWorkspace } from '../../widgets/ChatWorkspace/ChatWorkspace'
import styles from './AssistantPage.module.css'

export function AssistantPage() {
  return (
    <section className={styles.page}>
      <div className={styles.heading}>
        <span>ИИ-КОНСУЛЬТАНТ ENERGYTOOL</span>
        <h1>Опишите задачу</h1>
        <p>Расскажите об объекте, потреблении или проблеме своими словами. Консультант задаст уточняющий вопрос и предложит следующий шаг.</p>
      </div>
      <ChatWorkspace />
    </section>
  )
}
