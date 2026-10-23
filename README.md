# Все вместе

Веб-приложение для обмена запросами и предложениями. Главная страница ведёт к
форме; отправка сохраняет карточку в PostgreSQL и открывает её постоянную ссылку.

## Устройство

В Docker Compose запускаются **три контейнера**:

- `frontend` — Vue 3, собранный Vite, и nginx. Принимает HTTPS на порту **443**,
  отдаёт страницы и передаёт `/api/` в Go API. На **80** обслуживает только
  HTTP-01 проверки Let's Encrypt, остальные запросы перенаправляет на HTTPS (308).
- `api` — Go API на внутреннем порту 8080. Порт не публикуется на хост.
- `db` — PostgreSQL 17. Порт не публикуется на хост; данные хранятся в томе
  `sample-data`.

Код страниц находится в `frontend/src`, обработчики и хранилище —
в `sample-server`. Внешняя точка входа у страниц и API одна, поэтому отдельная
настройка CORS не нужна.

## Страницы и данные

| Адрес | Содержимое |
| --- | --- |
| `/` | Приветствие «Все вместе» и кнопка «Заполнить» |
| `/form` | Форма из пяти полей |
| `/view&id=1` | Карточка с ID 1; выводятся только заполненные поля формы |
| `/view?id=1` | Альтернативная стандартная запись ссылки на ту же карточку |
| `/admin` | Полная таблица карточек, включая ID и время создания |

Ссылка на отсутствующую карточку показывает «Не найдено». Страница `/admin`
и API доступны без авторизации; контакты в карточках также публичны.

Таблица `cards` создаётся при первом запуске API:

| Столбец | Тип и назначение |
| --- | --- |
| `id` | `BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY`, начинается с 1 |
| `timestamp` | `TIMESTAMPTZ`, время создания записи |
| `request` | `TEXT`, что нужно; обязательное поле |
| `offer` | `TEXT`, что есть или можно предложить |
| `details` | `TEXT`, где, когда и на каких условиях |
| `consideration` | `TEXT`, что пока неизвестно |
| `contact` | `TEXT`, имя и способ связи |

Остальные поля формы необязательны. Максимум — 10000 символов в каждом поле.
API использует JSON: `POST /api/cards`, `GET /api/cards`,
`GET /api/cards/{id}`. `GET /healthz` проверяет соединение с БД.

## Развёртывание на VDS с HTTPS

Нужны Git, запущенный Docker Engine с Docker Compose v2, домен и доступ к DNS.
Инструкции рассчитаны на Ubuntu/Debian и домен `vsevmeste.tech`.

### Если сертификат уже выпущен

Для подключения существующих файлов
`/etc/letsencrypt/live/vsevmeste.tech/fullchain.pem` и
`/etc/letsencrypt/live/vsevmeste.tech/privkey.pem` повторно выпускать сертификат
не нужно.

1. Проверьте DNS: запись **A** должна указывать на публичный IPv4 VDS. Если
   используется AAAA, она должна указывать на работающий IPv6 этого же сервера.
   Откройте входящие TCP-порты **80** и **443** в firewall VDS и панели провайдера.
   Оба порта должны быть свободны от других HTTP-серверов перед запуском Compose.
2. Клонируйте проект (либо обновите существующий клон) и создайте настройки,
   если `.env` ещё нет:

   ```sh
   git clone --branch work https://github.com/adudkin/sample.git
   cd sample
   cp .env.example .env
   ```

   У существующей установки отредактируйте текущий `.env`, не заменяя пароль БД:

   ```dotenv
   DOMAIN=vsevmeste.tech
   POSTGRES_PASSWORD=replace_with_a_unique_password
   TLS_DIRECTORY=/etc/letsencrypt
   ACME_WEBROOT=./certbot-www
   ```

   Не добавляйте `.env` и приватные ключи в Git. Пароль передаётся Go и
   PostgreSQL отдельной переменной, поэтому URL-кодирование не требуется.
   Смена переменной после инициализации тома сама по себе не меняет пароль БД.
3. Создайте каталог для HTTP-проверок. Его содержимое публично доступно только
   по пути `/.well-known/acme-challenge/`; приватные ключи туда не помещают:

   ```sh
   mkdir -p certbot-www/.well-known/acme-challenge
   chmod 755 certbot-www certbot-www/.well-known certbot-www/.well-known/acme-challenge
   ```

   Если в `.env` задан другой `ACME_WEBROOT`, создайте каталог по этому пути.
4. Запустите контейнеры и проверьте конфигурацию и готовность:

   ```sh
   docker compose up --build -d --remove-orphans
   docker compose ps
   docker compose exec -T frontend nginx -t
   curl --fail https://vsevmeste.tech/healthz
   curl -I 'http://vsevmeste.tech/form?source=check'
   ```

   Последняя команда должна показать `308` и
   `Location: https://vsevmeste.tech/form?source=check`.
   Откройте `https://vsevmeste.tech/`. Если API ещё запускается, повторите запрос
   готовности. При ошибке смотрите `docker compose logs --tail=100 frontend api db`.

В nginx подключаются `live/DOMAIN/fullchain.pem` и `live/DOMAIN/privkey.pem`.
В контейнер монтируется **весь** каталог `/etc/letsencrypt` только для чтения:
файлы в `live/` ссылаются на `archive/`. Сертификат не встраивается в образ.
Certbot работает на VDS и пишет файлы проверки в `ACME_WEBROOT`; nginx читает
этот каталог через отдельный read-only bind mount.

**Порт 80 не нужен для чтения уже выпущенного сертификата или самого TLS.**
Он используется для перенаправления на HTTPS и доступности HTTP-01 проверок
при продлении. Без действительных сертификата и ключа nginx не запустится;
порт 80 не включает приложение без HTTPS. Порты 8080 и 5432 на VDS открывать
не нужно.

### Если сертификата ещё нет

Сначала выпустите его через временный сервер Certbot, пока Compose ещё
**не запущен** и порт 80 свободен:

```sh
sudo apt-get update
sudo apt-get install -y certbot
sudo certbot certonly --standalone --preferred-challenges http \
  --cert-name vsevmeste.tech -d vsevmeste.tech \
  --email you@example.org --agree-tos
```

Укажите свою почту. Затем выполните шаги запуска выше и настройте продление
через webroot, как описано далее. Без сертификата TLS-сервер не стартует,
поэтому первая выдача и последующие продления имеют разный способ проверки.

### Продление без конфликта с портом 80

После запуска nginx порт 80 занят им, поэтому прежний метод Certbot
`standalone` нужно заменить на `webroot`. Останавливать сайт для продления
не потребуется.

Проверьте доступность каталога снаружи (из корня проекта):

```sh
printf 'acme-check\n' > certbot-www/.well-known/acme-challenge/check
chmod 644 certbot-www/.well-known/acme-challenge/check
curl --fail http://vsevmeste.tech/.well-known/acme-challenge/check
rm certbot-www/.well-known/acme-challenge/check
```

Ответ должен содержать `acme-check`, без перенаправления. Проверка отсутствующего
файла по этому пути возвращает 404. Остальные HTTP-адреса перенаправляются на
фиксированный домен из `DOMAIN`, с сохранением пути и query string.

Проверьте установленную версию Certbot и имя сертификата:

```sh
certbot --version
sudo certbot certificates
certbot help reconfigure
```

В **Certbot 2.3.0 и новее** смените метод проверки поддерживаемой командой:

```sh
sudo certbot reconfigure --cert-name vsevmeste.tech \
  --authenticator webroot --webroot-path /opt/sample/certbot-www
sudo certbot renew --cert-name vsevmeste.tech --dry-run
```

Замените `/opt/sample/certbot-www` абсолютным путём к каталогу из `ACME_WEBROOT`.
Если `certbot certificates` показывает другое имя сертификата, укажите его в
`--cert-name`. `reconfigure` сначала выполняет тестовое продление через staging
Let's Encrypt и при успехе сохраняет новые параметры для будущих продлений.
Один `renew --dry-run` проверяет параметры, но сам по себе их не сохраняет.
Если `reconfigure` недоступен, обновите Certbot поддерживаемым способом для своей
ОС; вручную редактировать файлы `/etc/letsencrypt/renewal/` не требуется.
См. [официальную документацию Certbot](https://eff-certbot.readthedocs.io/en/stable/using.html#modifying-the-renewal-configuration).

Проверьте автоматический запуск (для установки из Debian/Ubuntu-пакета):

```sh
sudo systemctl enable --now certbot.timer
systemctl list-timers certbot.timer
```

Для другой установки Certbot используйте её штатный таймер. Порт 80 должен
оставаться доступен из Интернета для будущих HTTP-01 проверок.

После успешного продления nginx должен перечитать сертификат. Создайте deploy-hook
`/etc/letsencrypt/renewal-hooks/deploy/reload-sample.sh` с содержимым:

```sh
#!/bin/sh
set -eu
cd /opt/sample
/usr/bin/docker compose exec -T frontend nginx -t
/usr/bin/docker compose exec -T frontend nginx -s reload
```

Замените `/opt/sample` абсолютным путём к клону, проверьте путь к Docker
командой `command -v docker` и сделайте hook исполняемым:

```sh
sudo chmod 750 /etc/letsencrypt/renewal-hooks/deploy/reload-sample.sh
```

Проверьте команды `nginx -t` и reload вручную из каталога проекта. Hook запускается
Certbot после успешного обновления, без перезапуска БД. Полный каталог
`/etc/letsencrypt` должен оставаться смонтированным, чтобы nginx увидел новые
файлы в `archive/` и обновлённые ссылки в `live/`.

## Локальная отладка — отключена по умолчанию

Базовый `docker-compose.yml` публикует HTTPS 443 и HTTP 80 для ACME/перенаправления.
Дополнительный файл `docker-compose.debug.yml` явно включает приложение по HTTP
**только на 127.0.0.1:8080**, без перенаправления. На публичном VDS этот файл не
подключайте.

Для локального запуска без реального домена нужны OpenSSL и свободные порты 80/443:

```sh
cp .env.example .env
./scripts/create-local-certificate.sh
mkdir -p certbot-www/.well-known/acme-challenge
chmod 755 certbot-www certbot-www/.well-known certbot-www/.well-known/acme-challenge
```

Укажите в `.env` `DOMAIN=localhost`, `TLS_DIRECTORY=./local-certs` и свой
`POSTGRES_PASSWORD`. Затем выполните:

```sh
docker compose -f docker-compose.yml -f docker-compose.debug.yml up --build -d
curl --fail --cacert local-certs/live/localhost/fullchain.pem https://localhost/healthz
curl --fail http://127.0.0.1:8080/healthz
```

Отладочная страница: `http://127.0.0.1:8080/`. Локальный сертификат
самоподписанный и действует 30 дней; для HTTPS в браузере добавьте его в доверенные
сертификаты своего устройства. Проверку TLS отключать не требуется.
Скрипт оставляет существующий сертификат без изменений.

Чтобы выключить локальную отладку по HTTP 8080, пересоздайте фронтенд с базовой
конфигурацией. HTTP 80 продолжит работать как ACME/redirect-вход:

```sh
docker compose up -d --force-recreate frontend
```

## Удаление старой БД перед переходом на карточки

Автоматического удаления или переноса старых текстов нет. Инициализация создаёт
новую таблицу `cards`; чтобы получить полностью чистую БД и начать ID с 1,
удалите старый том **перед первым запуском новой версии**.

Если нужна резервная копия, сделайте её, пока старая БД ещё работает. Команды
выполняются из каталога старого проекта (при необходимости используйте его
старую Compose-конфигурацию):

```sh
docker compose exec -T db pg_dump -U sample -d sample > sample-before-reset.sql
```

Затем из того же каталога и с тем же именем Compose-проекта выполните:

```sh
docker compose down --volumes --remove-orphans
docker compose up --build -d
```

**`down --volumes` необратимо удаляет данные БД этого проекта.** Для каталога
`sample` по умолчанию удаляется том `sample_sample-data`. При переименовании
каталога или использовании `-p`/`COMPOSE_PROJECT_NAME` имя будет другим;
проверьте его через `docker volume ls` и `docker volume inspect` до удаления.
Не удаляйте все тома Docker. Сертификаты, подключённые как bind mount, эта
команда не удаляет.

Обычная остановка сохраняет БД:

```sh
docker compose down
```

## Проверки разработки

Проверка HTTP-редиректа, HTTPS с проверкой сертификата, доступности приложения
и выдачи ACME-файла (из каталога проекта на VDS):

```sh
DOMAIN=vsevmeste.tech ACME_WEBROOT=./certbot-www python3 tests/https.py
```

Проверка не меняет записи БД. В каталоге ACME создаётся временный файл,
который удаляется после запроса. Без `ACME_WEBROOT` эта проверка файла
будет явно пропущена. Для локального тестового сертификата также задайте
`CA_CERT=local-certs/live/localhost/fullchain.pem` и `DOMAIN=localhost`.

При запущенном приложении интеграционные проверки:

```sh
BASE_URL=https://vsevmeste.tech python3 tests/smoke.py
```

Для локального самоподписанного сертификата передайте доверенный CA, сохраняя
проверку TLS:

```sh
CA_CERT=local-certs/live/localhost/fullchain.pem \
  BASE_URL=https://localhost python3 tests/smoke.py
```

Для локального HTTP используйте `BASE_URL=http://127.0.0.1:8080`.
Тесты записи создают тестовые карточки в выбранной БД.

Необязательная проверка интерфейса в настоящем браузере:

```sh
python3 -m venv .local/browser-venv
.local/browser-venv/bin/pip install playwright
.local/browser-venv/bin/playwright install chromium
BASE_URL=http://127.0.0.1:8080 .local/browser-venv/bin/python tests/browser.py
```

Если Chromium уже установлен в системе, вместо скачивания задайте путь к нему:

```sh
CHROMIUM_PATH=/usr/bin/chromium BASE_URL=http://127.0.0.1:8080 \
  .local/browser-venv/bin/python tests/browser.py
```

Для запуска браузерной проверки через HTTPS сертификат должен быть доверенным
в самом браузере. `CA_CERT` настраивает только Python smoke-тест, а не Chromium.

Go-проверки из `sample-server` (Go 1.24+):

```sh
go mod download
go test ./...
```

Сборка Vue из `frontend` (Node.js 22.12+ или 24+, npm):

```sh
npm ci
npm run build
```

### Облачный builder с доверенным HTTPS-прокси

Оба Dockerfile поддерживают необязательный BuildKit secret `cloud_ca` с доверенным
CA-bundle. Передайте его только на время сборки; TLS и проверки зависимостей
остаются включёнными. Это отдельная настройка инфраструктуры сборки, а не
требование обычного VDS. Приватные ключи сертификатов сайта для неё не нужны.

## Оформление и контраст

Структура экранов повторяет компактную форму из прототипа: шапка, карточка
с полями и пояснениями, отдельное приветствие. Палитра заменена на зелёную
и кремовую, шрифт — без засечек. На телефоне основной текст и поля от 16 px,
кнопки от 46 px, форма занимает одну колонку.

Контраст проверен отдельно по формуле WCAG: белый текст на тёмно-зелёной
кнопке — 9,74:1; основной текст на белом — 12,44:1; пояснения на светло-зелёном
блоке — 5,74:1; текст на кремовом — 8,12:1; ошибки — не ниже 6,65:1.
Граница поля на светло-зелёном блоке — 3,53:1. Ошибки дополнительно
обозначаются словом «Ошибка», а обязательность — текстом у подписи.
