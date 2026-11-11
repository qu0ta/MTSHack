import { FormEvent, useState } from 'react'
import { validateLead, type LeadData, type LeadErrors } from '../../features/lead/model/lead'
import styles from './LeadForm.module.css'

interface LeadFormProps {
  summary: string
  isComplete: boolean
  onComplete: () => void
  onClose: () => void
}

const emptyLead: LeadData = { name: '', contact: '', consent: false }

export function LeadForm({ summary, isComplete, onComplete, onClose }: LeadFormProps) {
  const [lead, setLead] = useState(emptyLead)
  const [errors, setErrors] = useState<LeadErrors>({})

  const submit = (event: FormEvent) => {
    event.preventDefault()
    const nextErrors = validateLead(lead)
    setErrors(nextErrors)
    if (Object.keys(nextErrors).length === 0) onComplete()
  }

  return (
    <div className={styles.overlay} role="dialog" aria-modal="true" aria-labelledby="lead-title">
      <div className={styles.top}>
        <span>HANDOFF / MANAGER</span>
        <button type="button" onClick={onClose} aria-label="Закрыть форму">×</button>
      </div>

      {isComplete ? (
        <div className={styles.success}>
          <span>✓</span>
          <h2 id="lead-title" aria-label="Заявка собрана">Заявка<br />собрана</h2>
          <p>В рабочем продукте контакт и резюме будут отправлены менеджеру.</p>
          <div><b>РЕЗЮМЕ</b>{summary || 'Клиент запросил консультацию по продуктам EnergyTool.'}</div>
          <button type="button" onClick={onClose}>Вернуться к диалогу</button>
        </div>
      ) : (
        <form className={styles.form} onSubmit={submit} noValidate>
          <span>[ СЛЕДУЮЩИЙ ШАГ ]</span>
          <h2 id="lead-title" aria-label="Оставить контакт">Оставить<br />контакт</h2>
          <label>Как к вам обращаться
            <input value={lead.name} onChange={(event) => setLead({ ...lead, name: event.target.value })} aria-invalid={Boolean(errors.name)} />
            {errors.name && <small>{errors.name}</small>}
          </label>
          <label>Телефон или email
            <input value={lead.contact} onChange={(event) => setLead({ ...lead, contact: event.target.value })} aria-invalid={Boolean(errors.contact)} />
            {errors.contact && <small>{errors.contact}</small>}
          </label>
          <label className={styles.consent}>
            <input type="checkbox" checked={lead.consent} onChange={(event) => setLead({ ...lead, consent: event.target.checked })} />
            <span>Согласен на обработку данных для обратной связи</span>
          </label>
          {errors.consent && <small>{errors.consent}</small>}
          <button type="submit">Сформировать заявку <b>↗</b></button>
        </form>
      )}
    </div>
  )
}
