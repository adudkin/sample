<!-- Страница заполнения формы -->
<script setup>
import { nextTick, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { createCard } from '../lib/api.js'
import { fields, MAX_FIELD_LENGTH } from '../lib/fields.js'

const router = useRouter()
const values = reactive(Object.fromEntries(fields.map((field) => [field.name, ''])))
const errors = ref({})
const saving = ref(false)
const status = ref('')

function validate() {
  const result = {}
  for (const field of fields) {
    if (field.required && !values[field.name].trim()) {
      result[field.name] = 'Укажите, что вам нужно.'
    } else if ([...values[field.name]].length > MAX_FIELD_LENGTH) {
      result[field.name] = `Сократите текст до ${MAX_FIELD_LENGTH} символов.`
    }
  }
  return result
}

async function focusFirstError() {
  await nextTick()
  document.getElementById(Object.keys(errors.value)[0])?.focus()
}

async function submit() {
  if (saving.value) return
  errors.value = validate()
  status.value = ''
  if (Object.keys(errors.value).length) {
    status.value = 'Проверьте поля с пометкой «Ошибка».'
    await focusFirstError()
    return
  }
  saving.value = true
  try {
    const card = await createCard(values)
    await router.push(`/view&id=${card.id}`)
  } catch (error) {
    errors.value = error.fields || {}
    status.value = error instanceof TypeError
      ? 'Не удалось связаться с сервером. Проверьте соединение и повторите отправку.'
      : error.message
    if (Object.keys(errors.value).length) await focusFirstError()
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <section class="panel form-panel" aria-labelledby="form-title">
    <p class="eyebrow">Ваша карточка</p>
    <h1 id="form-title">Расскажите о своей цели</h1>
    <p class="intro">Запишите, что вам нужно и чем вы готовы поделиться. Начните с главного; остальные поля можно оставить пустыми.</p>
    <p class="notice">Карточка сохраняется после нажатия «Отправить». Все записи доступны в админке без авторизации — не указывайте закрытые сведения.</p>
    <form id="card-form" novalidate @submit.prevent="submit">
      <div v-for="field in fields" :key="field.name" class="field">
        <label :for="field.name">
          {{ field.label }}
          <span v-if="field.required" class="required-note">(обязательно)</span>
        </label>
        <textarea
          :id="field.name"
          v-model="values[field.name]"
          :name="field.name"
          :required="field.required"
          :disabled="saving"
          :aria-invalid="errors[field.name] ? 'true' : undefined"
          :aria-describedby="`${field.name}-help${errors[field.name] ? ` ${field.name}-error` : ''}`"
          rows="3"
        ></textarea>
        <p :id="`${field.name}-help`" class="field-help">{{ field.help }}</p>
        <p v-if="errors[field.name]" :id="`${field.name}-error`" class="field-error">Ошибка: {{ errors[field.name] }}</p>
      </div>
      <p v-if="status" class="message message-error" role="alert">{{ status }}</p>
      <div class="form-footer">
        <p class="field-help">До {{ MAX_FIELD_LENGTH.toLocaleString('ru-RU') }} символов в каждом поле.</p>
        <button class="button button-primary" type="submit" :disabled="saving" :aria-busy="saving">{{ saving ? 'Отправляем…' : 'Отправить' }}</button>
      </div>
    </form>
  </section>
</template>
