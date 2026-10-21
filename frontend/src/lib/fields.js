export const MAX_FIELD_LENGTH = 10000

export const fields = [
  {
    name: 'request',
    label: 'Что мне нужно',
    help: 'Вещь, работа, возможность или другой нужный результат. Указать количество, если оно имеет смысл.',
    required: true,
  },
  {
    name: 'offer',
    label: 'Что у меня есть или что могу предложить',
    help: 'Реальные вещи, умения, время, деньги или действия. Несколько ресурсов записать отдельно.',
  },
  {
    name: 'details',
    label: 'Где, когда и на каких условиях',
    help: 'Место, срок, состояние, допустимые варианты и ограничения.',
  },
  {
    name: 'consideration',
    label: 'Что пока не знаю',
    help: 'Условия, которые нельзя честно заполнить без уточнения.',
  },
  {
    name: 'contact',
    label: 'Имя и способ связи',
    help: 'Как обращаться и каким способом задать вопрос или получить ответ.',
  },
]

export function formatTimestamp(value) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('ru-RU')
}
