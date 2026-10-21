export class APIError extends Error {
  constructor(message, status, fields = {}) {
    super(message)
    this.status = status
    this.fields = fields
  }
}

async function request(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...options.headers },
  })
  let body
  try {
    body = await response.json()
  } catch {
    throw new APIError('Сервер вернул непонятный ответ. Попробуйте ещё раз.', response.status)
  }
  if (!response.ok) {
    throw new APIError(body.error || 'Не удалось выполнить запрос.', response.status, body.fields)
  }
  return body
}

export const createCard = (card) => request('/api/cards', { method: 'POST', body: JSON.stringify(card) })
export const getCard = (id, signal) => request(`/api/cards/${encodeURIComponent(id)}`, { signal })
export const getCards = (signal) => request('/api/cards', { signal })
