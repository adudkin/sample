import { nextTick } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import WelcomePage from './pages/WelcomePage.vue'
import FormPage from './pages/FormPage.vue'
import CardPage from './pages/CardPage.vue'
import AdminPage from './pages/AdminPage.vue'
import NotFoundPage from './pages/NotFoundPage.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: WelcomePage, meta: { title: 'Все вместе' } },
    { path: '/form', component: FormPage, meta: { title: 'Заполнить карточку' } },
    // The original requested URL uses an ampersand in the path, not a query.
    { path: '/view&id=:id', component: CardPage, meta: { title: 'Карточка' } },
    { path: '/view', component: CardPage, meta: { title: 'Карточка' } },
    { path: '/admin', component: AdminPage, meta: { title: 'Панель администрирования' } },
    { path: '/:pathMatch(.*)*', component: NotFoundPage, meta: { title: 'Не найдено' } },
  ],
  scrollBehavior: () => ({ top: 0 }),
})

router.afterEach(async (to) => {
  document.title = to.meta.title === 'Все вместе' ? 'Все вместе' : `${to.meta.title} — Все вместе`
  await nextTick()
  document.querySelector('main')?.focus({ preventScroll: true })
})

export default router
