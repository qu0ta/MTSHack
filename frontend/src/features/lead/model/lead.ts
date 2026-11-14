export type LeadData = { name: string; phone: string; company: string; consent: boolean }
export type LeadErrors = Partial<Record<keyof LeadData, string>>

export function formatPhone(value: string): string {
  if (!value.trim()) return ''
  let digits = value.replace(/\D/g, '')
  if (!digits) return ''
  if (value.trim().startsWith('+7') || (digits.length === 11 && (digits[0] === '7' || digits[0] === '8'))) digits = digits.slice(1)
  digits = digits.slice(0, 10)
  let result = '+7'
  if (digits.length) result += ` (${digits.slice(0, 3)}`
  if (digits.length >= 3) result += ')'
  if (digits.length > 3) result += ` ${digits.slice(3, 6)}`
  if (digits.length > 6) result += `-${digits.slice(6, 8)}`
  if (digits.length > 8) result += `-${digits.slice(8, 10)}`
  return result
}

export function validateLead(lead: LeadData): LeadErrors {
  const errors: LeadErrors = {}
  if (!lead.name.trim()) errors.name = 'Укажите имя'
  else if (Array.from(lead.name.trim()).length > 255) errors.name = 'Не больше 255 символов'
  if (!lead.company.trim()) errors.company = 'Укажите компанию'
  else if (Array.from(lead.company.trim()).length > 255) errors.company = 'Не больше 255 символов'
  const phone = lead.phone.trim()
  if (!phone || phone === '+7') errors.phone = 'Укажите телефон'
  else if (!/^\+7 \(\d{3}\) \d{3}-\d{2}-\d{2}$/.test(phone)) errors.phone = 'Введите номер полностью'
  if (!lead.consent) errors.consent = 'Нужно согласие на обработку данных'
  return errors
}
