import { describe, expect, it } from 'vitest'
import { buildLeadSummary, validateLead } from './lead'

describe('validateLead', () => {
  it('принимает имя, телефон и согласие', () => {
    expect(validateLead({ name: 'Анна', contact: '+7 900 000-00-00', consent: true })).toEqual({})
  })

  it('возвращает ошибки для пустых полей и отсутствующего согласия', () => {
    expect(validateLead({ name: '', contact: '', consent: false })).toEqual({
      name: 'Укажите имя',
      contact: 'Укажите телефон или email',
      consent: 'Нужно согласие на обработку данных',
    })
  })

  it('отклоняет контакт, который не похож на телефон или email', () => {
    expect(validateLead({ name: 'Анна', contact: 'позвоните мне', consent: true })).toEqual({
      contact: 'Проверьте телефон или email',
    })
  })
})

describe('buildLeadSummary', () => {
  it('собирает резюме из последних реплик клиента', () => {
    const summary = buildLeadSummary([
      { id: '1', role: 'user', text: 'Хотим поставить солнечные панели', createdAt: 1 },
      { id: '2', role: 'assistant', text: 'Какой регион?', createdAt: 2 },
      { id: '3', role: 'user', text: 'Объект в Екатеринбурге', createdAt: 3 },
    ])

    expect(summary).toBe('Хотим поставить солнечные панели · Объект в Екатеринбурге')
  })
})
