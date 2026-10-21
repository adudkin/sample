<!-- админка -->
<script setup>
import { onBeforeUnmount, ref } from 'vue'
import { getCards } from '../lib/api.js'
import { fields, formatTimestamp } from '../lib/fields.js'

const cards = ref([])
const loading = ref(false)
const error = ref('')
let controller

async function load() {
  controller?.abort()
  controller = new AbortController()
  loading.value = true
  error.value = ''
  try {
    cards.value = await getCards(controller.signal)
  } catch (failure) {
    if (failure.name !== 'AbortError') error.value = 'Не удалось загрузить записи. Нажмите «Обновить» для повторной попытки.'
  } finally {
    loading.value = false
  }
}

onBeforeUnmount(() => controller?.abort())
load()
</script>

<template>
  <section class="panel admin-panel" aria-labelledby="admin-title">
    <p class="eyebrow">Все записи</p>
    <h1 id="admin-title">Панель администрирования</h1>
    <p class="intro">Полная таблица карточек, новые записи сверху. Раздел доступен без авторизации.</p>
    <button class="button button-secondary" type="button" :disabled="loading" @click="load">{{ loading ? 'Загружаем…' : 'Обновить' }}</button>
    <p v-if="error" class="message message-error" role="alert">{{ error }}</p>
    <p v-else-if="loading" role="status">Загружаем записи…</p>
    <p v-else-if="!cards.length" class="empty-state" role="status">Карточек пока нет.</p>
    <template v-else>
      <p class="table-hint" id="table-hint">Всего карточек: {{ cards.length }}. Таблицу можно прокручивать по горизонтали.</p>
      <div class="table-scroll" tabindex="0" role="region" aria-label="Таблица всех карточек" aria-describedby="table-hint">
        <table class="admin-table">
          <caption class="visually-hidden">Все поля таблицы карточек в базе данных</caption>
          <thead><tr><th scope="col">id</th><th scope="col">timestamp</th><th v-for="field in fields" :key="field.name" scope="col">{{ field.label }}<span class="column-name">{{ field.name }}</span></th></tr></thead>
          <tbody>
            <tr v-for="card in cards" :key="card.id">
              <th scope="row"><RouterLink :to="`/view&id=${card.id}`">{{ card.id }}</RouterLink></th>
              <td><time :datetime="card.timestamp">{{ formatTimestamp(card.timestamp) }}</time></td>
              <td v-for="field in fields" :key="field.name">{{ card[field.name] || '—' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </section>
</template>
