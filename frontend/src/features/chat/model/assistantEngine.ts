export type AssistantTopic = 'solar' | 'metering' | 'losses' | 'wholesale' | 'unknown'

export interface ChatHistoryItem {
  role: 'user' | 'assistant'
  text: string
}

export interface AssistantReply {
  text: string
  topic: AssistantTopic
  sources: string[]
  responseTimeMs: number
}

const includesAny = (value: string, patterns: string[]) => patterns.some((pattern) => value.includes(pattern))

export function getAssistantReply(message: string, history: ChatHistoryItem[]): AssistantReply {
  const normalized = message.trim().toLocaleLowerCase('ru-RU')

  if (!normalized) {
    throw new Error('Введите сообщение')
  }

  const responseTimeMs = 680 + Math.min(normalized.length * 7 + history.length * 31, 760)

  if (includesAny(normalized, ['солнеч', 'панел', 'крыш', 'кровл', 'инвертор'])) {
    return {
      topic: 'solar',
      text: 'Солнечную станцию стоит оценивать по профилю потребления, характеристикам кровли и месту установки. Точный эффект и стоимость определяются только расчётом по объекту. В каком регионе находится объект?',
      sources: ['23-solar.md', '03-calculation-and-requests.md'],
      responseTimeMs,
    }
  }

  if (includesAny(normalized, ['счётчик', 'счетчик', 'показани', 'учёт', 'учет', 'аскуэ'])) {
    return {
      topic: 'metering',
      text: 'Для автоматического учёта сначала нужно понять состав оборудования и текущий способ сбора данных. Платформа может работать с разными схемами передачи показаний, но совместимость проверяется отдельно. Какие модели счётчиков установлены на объектах?',
      sources: ['22-askue.md', '03-calculation-and-requests.md'],
      responseTimeMs,
    }
  }

  if (includesAny(normalized, ['потер', 'хищен', 'небаланс'])) {
    return {
      topic: 'losses',
      text: 'Поиск коммерческих потерь предназначен для сетевых организаций, а не для обычных промышленных предприятий или энергосбытовых компаний. Результат - список приоритетных проверок, а не доказанный факт хищения. Ваша организация относится к электросетевым?',
      sources: ['24-losses.md', '03-calculation-and-requests.md'],
      responseTimeMs,
    }
  }

  if (includesAny(normalized, ['оптов', 'орэм', 'мощност', '750 квт', '20 мвт'])) {
    return {
      topic: 'wholesale',
      text: 'При мощности ниже 750 кВт выход на оптовый рынок невозможен. В диапазоне от 750 кВт до 20 МВт участие возможно через энергосбытовую компанию-посредника, а от 20 МВт - напрямую. Какая присоединённая мощность у вашего объекта?',
      sources: ['25-wholesale-market.md', '03-calculation-and-requests.md'],
      responseTimeMs,
    }
  }

  return {
    topic: 'unknown',
    text: 'Я консультирую только по продуктам EnergyTool: учёту электроэнергии, солнечным станциям, поиску коммерческих потерь и выходу на оптовый рынок. Опишите энергетическую задачу вашего бизнеса - я помогу определить следующий шаг.',
    sources: ['00-assistant-scope.md'],
    responseTimeMs,
  }
}
