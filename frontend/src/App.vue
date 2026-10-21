<template>
  <a class="skip-link" href="#main">К содержимому</a>
  <header class="topbar">
    <RouterLink class="brand" to="/" aria-label="Все вместе — главная">
      <span class="brand-mark" aria-hidden="true">В</span>
      <span>Все вместе</span>
    </RouterLink>
    <nav aria-label="Основная навигация">
      <RouterLink to="/form">Заполнить</RouterLink>
      <RouterLink to="/admin">Админка</RouterLink>
    </nav>
  </header>
  <main id="main" tabindex="-1">
    <RouterView />
  </main>
  <footer class="site-footer">Общие усилия. Новые возможности.</footer>
</template>
