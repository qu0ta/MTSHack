import { useEffect, useRef, useState, type FormEvent } from 'react'
import { validateLead, type LeadData, type LeadErrors } from '../../features/lead/model/lead'
import type { Application } from '../../features/chat/api/assistantService'
import styles from './LeadForm.module.css'

type Props = {
  application: Application | null
  pending: boolean
  error: string
  onSubmit: (lead: LeadData) => void
  onClose: () => void
  onRefresh: () => void
  uncertain: boolean
}

export function LeadForm({ application, pending, error, uncertain, onSubmit, onClose, onRefresh }: Props) {
  const [lead, setLead] = useState<LeadData>({ name: application?.full_name ?? '', phone: application?.phone ?? '', company: application?.company ?? '', consent: false })
  const [errors, setErrors] = useState<LeadErrors>({})
  const dialog = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null
    dialog.current?.querySelector<HTMLButtonElement>('button')?.focus()
    return () => previous?.focus()
  }, [])

  const submit = (event: FormEvent) => {
    event.preventDefault()
    const next = validateLead(lead)
    setErrors(next)
    if (!Object.keys(next).length) onSubmit(lead)
  }

  return (
    <div ref={dialog} className={styles.overlay} role="dialog" aria-modal="true" aria-labelledby="lead-title" onKeyDown={(event) => {
      if (event.key === 'Escape' && !pending) onClose()
      if (event.key !== 'Tab') return
      const controls = Array.from(dialog.current?.querySelectorAll<HTMLElement>('button:not(:disabled), input:not(:disabled)') ?? [])
      if (!controls.length) return
      if (event.shiftKey && document.activeElement === controls[0]) { event.preventDefault(); controls[controls.length - 1].focus() }
      else if (!event.shiftKey && document.activeElement === controls[controls.length - 1]) { event.preventDefault(); controls[0].focus() }
    }}>
      <div className={styles.top}>
        <span>ЗАЯВКА МЕНЕДЖЕРУ</span>
        <button type="button" disabled={pending} onClick={onClose} aria-label="Закрыть форму">×</button>
      </div>
      {application ? (
        <div className={styles.success}>
          <span>✓</span><h2 id="lead-title">Заявка сохранена</h2>
          <p>Данные сохранены в системе. Отправка по почте или в CRM не подтверждена.</p>
          <div><b>РЕЗЮМЕ</b>{application.summary}</div>
          <button type="button" onClick={onClose}>Вернуться к диалогу</button>
        </div>
      ) : (
        <form className={styles.form} onSubmit={submit} noValidate>
          <h2 id="lead-title">Оставить контакт</h2>
          <label>Как к вам обращаться
            <input value={lead.name} onChange={(event) => setLead({ ...lead, name: event.target.value })} aria-invalid={Boolean(errors.name)} aria-describedby={errors.name ? 'lead-name-error' : undefined} />
            {errors.name && <small id="lead-name-error">{errors.name}</small>}
          </label>
          <label>Телефон
            <input type="tel" value={lead.phone} onChange={(event) => setLead({ ...lead, phone: event.target.value })} aria-invalid={Boolean(errors.phone)} aria-describedby={errors.phone ? 'lead-phone-error' : undefined} />
            {errors.phone && <small id="lead-phone-error">{errors.phone}</small>}
          </label>
          <label>Компания
            <input value={lead.company} onChange={(event) => setLead({ ...lead, company: event.target.value })} aria-invalid={Boolean(errors.company)} aria-describedby={errors.company ? 'lead-company-error' : undefined} />
            {errors.company && <small id="lead-company-error">{errors.company}</small>}
          </label>
          <label className={styles.consent}>
            <input type="checkbox" checked={lead.consent} onChange={(event) => setLead({ ...lead, consent: event.target.checked })} aria-describedby={errors.consent ? 'lead-consent-error' : undefined} />
            <span>Согласен на обработку данных для обратной связи</span>
          </label>
          {errors.consent && <small id="lead-consent-error">{errors.consent}</small>}
          {error && <p role="alert">{error}</p>}
          {uncertain && <button type="button" disabled={pending} onClick={onRefresh}>Проверить историю и заявку</button>}
          <button type="submit" disabled={pending || uncertain}>{pending ? 'Сохраняем...' : 'Сохранить заявку'} <b>↗</b></button>
        </form>
      )}
    </div>
  )
}
