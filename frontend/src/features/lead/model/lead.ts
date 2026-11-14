export type LeadData = { name: string; phone: string; company: string; consent: boolean }
export type LeadErrors = Partial<Record<keyof LeadData, string>>

export function validateLead(lead: LeadData): LeadErrors {
  const errors: LeadErrors = {}
  if (!lead.name.trim()) errors.name = 'Укажите имя'
  else if (Array.from(lead.name.trim()).length > 255) errors.name = 'Не больше 255 символов'
  if (!lead.company.trim()) errors.company = 'Укажите компанию'
  else if (Array.from(lead.company.trim()).length > 255) errors.company = 'Не больше 255 символов'
  const phone = lead.phone.trim()
  if (!phone) errors.phone = 'Укажите телефон'
  else if (Array.from(phone).length > 30 || !/^\+?[\d\s()-]+$/.test(phone) || phone.replace(/\D/g, '').length < 10) errors.phone = 'Проверьте номер телефона'
  if (!lead.consent) errors.consent = 'Нужно согласие на обработку данных'
  return errors
}
