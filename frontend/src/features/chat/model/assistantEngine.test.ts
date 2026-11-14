import { describe, expect, it } from 'vitest'
import { getAssistantReply } from './assistantEngine'

describe('getAssistantReply', () => {
  it('подбирает солнечную станцию и уточняет регион', () => {
    const result = getAssistantReply('Хотим поставить солнечные панели на крышу производства', [])

    expect(result.topic).toBe('solar')
    expect(result.text.toLocaleLowerCase('ru-RU')).toContain('солнеч')
    expect(result.text).toContain('регион')
    expect(result.sources).toEqual(['23-solar.md', '03-calculation-and-requests.md'])
  })

  it('не рекомендует поиск потерь обычному предприятию', () => {
    const result = getAssistantReply('У нас обычное производство. Найдите коммерческие потери', [])

    expect(result.topic).toBe('losses')
    expect(result.text).toContain('сетевых организаций')
  })

  it('уточняет модели счётчиков для автоматического учёта', () => {
    const result = getAssistantReply('Хотим автоматически собирать показания с сорока счётчиков', [])

    expect(result.topic).toBe('metering')
    expect(result.text).toContain('модели')
    expect(result.sources).toContain('22-askue.md')
  })

  it('объясняет порог выхода на оптовый рынок', () => {
    const result = getAssistantReply('Мощность 600 кВт, можем выйти на оптовый рынок?', [])

    expect(result.topic).toBe('wholesale')
    expect(result.text).toContain('750 кВт')
    expect(result.text).toContain('невозможен')
  })

  it('не выдумывает ответ за пределами EnergyTool', () => {
    const result = getAssistantReply('Составь план найма разработчиков', [])

    expect(result.topic).toBe('unknown')
    expect(result.text).toContain('только по продуктам EnergyTool')
  })

  it('отклоняет пустой ввод', () => {
    expect(() => getAssistantReply('   ', [])).toThrow('Введите сообщение')
  })
})
