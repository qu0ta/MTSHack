import { describe, expect, it } from 'vitest'
import { validateLead } from './lead'

describe('validateLead', () => {
  it('принимает имя, компанию, телефон и согласие', () => {
    expect(validateLead({ name: 'Анна', company: 'ООО Тест', phone: '+7 900 000-00-00', consent: true })).toEqual({})
  })

  it('требует заполнить контактные данные и согласие', () => {
    expect(validateLead({ name: '', company: '', phone: '', consent: false })).toEqual({ name: 'Укажите имя', company: 'Укажите компанию', phone: 'Укажите телефон', consent: 'Нужно согласие на обработку данных' })
  })

  it('не принимает email в поле телефона', () => {
    expect(validateLead({ name: 'Анна', company: 'ООО Тест', phone: 'user@example.com', consent: true }).phone).toBe('Проверьте номер телефона')
  })
})
