<!-- страница просмотра карточки -->
<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { getCard } from '../lib/api.js'
import { fields } from '../lib/fields.js'

const route = useRoute()
const card = ref(null)
const loading = ref(false)
const error = ref('')
const notFound = ref(false)
const filledFields = computed(() => fields.filter((field) => card.value?.[field.name]?.trim()))

watch(() => route.fullPath, async (_, previous, onCleanup) => {
  const controller = new AbortController()
  onCleanup(() => controller.abort())
  card.value = null
  loading.value = false
  error.value = ''
  notFound.value = false
  const id = route.params.id || route.query.id
  if (typeof id !== 'string' || !/^[1-9]\d*$/.test(id)) {
    notFound.value = true
    return
  }
  loading.value = true
  try {
    card.value = await getCard(id, controller.signal)
  } catch (failure) {
    if (failure.name === 'AbortError') return
    if (failure.status === 404) notFound.value = true
    else error.value = 'Не удалось загрузить карточку. Попробуйте обновить страницу.'
  } finally {
    if (!controller.signal.aborted) loading.value = false
  }
}, { immediate: true })
</script>

<template>
  <section class="panel" aria-labelledby="card-title">
    <p class="eyebrow">Результат заполнения</p>
    <h1 id="card-title">{{ notFound ? 'Не найдено' : 'Карточка' }}</h1>
    <p v-if="loading" role="status">Загружаем карточку…</p>
    <p v-else-if="notFound" class="intro">Карточки с таким адресом нет.</p>
    <p v-else-if="error" class="message message-error" role="alert">{{ error }}</p>
    <table v-else-if="card" class="card-table">
      <caption class="visually-hidden">Заполненные поля карточки</caption>
      <thead><tr><th scope="col">Имя</th><th scope="col">Значение</th></tr></thead>
      <tbody>
        <tr v-for="field in filledFields" :key="field.name">
          <th scope="row">{{ field.label }}</th>
          <td>{{ card[field.name] }}</td>
        </tr>
      </tbody>
    </table>
    <div class="page-actions">
      <RouterLink class="button button-primary" to="/form">Заполнить новую карточку</RouterLink>
      <RouterLink class="text-link" to="/">На главную</RouterLink>
    </div>
  </section>
</template>
