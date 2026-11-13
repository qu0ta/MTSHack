import type { ChatMessage } from '../../chat/model/chatReducer'

export interface LeadData {
  name: string
  contact: string
  consent: boolean
}

export type LeadErrors = Partial<Record<keyof LeadData, string>>

export function validateLead(lead: LeadData): LeadErrors {
  const errors: LeadErrors = {}
  const contact = lead.contact.trim()
  const isEmail = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(contact)
  const isPhone = contact.replace(/\D/g, '').length >= 10

  if (!lead.name.trim()) errors.name = 'Укажите имя'
  if (!contact) errors.contact = 'Укажите телефон или email'
  else if (!isEmail && !isPhone) errors.contact = 'Проверьте телефон или email'
  if (!lead.consent) errors.consent = 'Нужно согласие на обработку данных'

  return errors
}

export function buildLeadSummary(messages: ChatMessage[]): string {
  return messages
    .filter((message) => message.role === 'user')
    .slice(-3)
    .map((message) => message.text.trim())
    .filter(Boolean)
    .join(' · ')
}
