# Все вместе

Веб-приложение для обмена запросами и предложениями. Главная страница ведёт к
форме; отправка сохраняет карточку в PostgreSQL и открывает её постоянную ссылку.

## Устройство

В Docker Compose запускаются **три контейнера**:

- `frontend` — Vue 3, собранный Vite, и nginx. Принимает HTTPS на порту **443**,
  отдаёт страницы и передаёт `/api/` в Go API.
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
Команды ниже рассчитаны на Ubuntu/Debian; замените `example.org` своим доменом,
а `you@example.org` — своей почтой.

1. Создайте DNS-запись **A** домена с публичным IPv4 VDS. Если используете AAAA,
   она должна указывать на работающий IPv6 этого же сервера. Откройте входящие
   TCP-порты **443** и **80** в firewall VDS и в панели провайдера. Порт 80
   нужен для проверки Let's Encrypt, приложение слушает только 443.
2. Получите сертификат. Убедитесь, что порт 80 свободен: режим `standalone`
   временно запускает на нём сервер проверки.

   ```sh
   sudo apt-get update
   sudo apt-get install -y certbot
   sudo certbot certonly --standalone --preferred-challenges http \
     -d example.org --email you@example.org --agree-tos
   ```

3. Клонируйте проект и создайте локальный файл настроек:

   ```sh
   git clone --branch work https://github.com/adudkin/sample.git
   cd sample
   cp .env.example .env
   ```

   Отредактируйте `.env` до первого запуска:

   ```dotenv
   DOMAIN=example.org
   POSTGRES_PASSWORD=replace_with_a_unique_password
   TLS_DIRECTORY=/etc/letsencrypt
   ```

   Не добавляйте `.env` и приватные ключи в Git. Пароль передаётся Go и
   PostgreSQL отдельной переменной, поэтому URL-кодирование не требуется.
   Смена переменной после инициализации тома сама по себе не меняет пароль БД.

4. Запустите контейнеры:

   ```sh
   docker compose up --build -d --remove-orphans
   docker compose ps
   curl --fail https://example.org/healthz
   ```

   Откройте `https://example.org/`. Если API ещё запускается, повторите запрос
   готовности. При ошибке проверьте `docker compose logs --tail=100 frontend api db`.

В nginx подключаются `live/DOMAIN/fullchain.pem` и `live/DOMAIN/privkey.pem`.
В контейнер монтируется **весь** каталог `/etc/letsencrypt` только для чтения:
файлы в `live/` ссылаются на `archive/`. Сертификат не встраивается в образ.
Без сертификата фронтенд не запустится; небезопасный HTTP автоматически не
включается. На VDS порты 8080 и 5432 открывать не нужно.

### Автоматическое продление сертификата

Проверьте таймер certbot и тестовое продление:

```sh
sudo systemctl enable --now certbot.timer
sudo certbot renew --dry-run
```

Порт 80 должен оставаться доступен для будущих HTTP-проверок. После успешного
продления nginx должен перечитать сертификат. Создайте deploy-hook
`/etc/letsencrypt/renewal-hooks/deploy/reload-sample.sh` с содержимым:

```sh
#!/bin/sh
set -eu
cd /opt/sample
/usr/bin/docker compose exec -T frontend nginx -s reload
```

Замените `/opt/sample` абсолютным путём к своему клону, проверьте путь к Docker
командой `command -v docker` и сделайте hook исполняемым:

```sh
sudo chmod 750 /etc/letsencrypt/renewal-hooks/deploy/reload-sample.sh
```

Сначала можно проверить команду reload вручную из каталога проекта. Hook
запускается certbot после успешного обновления, без перезапуска БД.

## Локальная отладка — отключена по умолчанию

Базовый `docker-compose.yml` публикует только HTTPS 443. Дополнительный файл
`docker-compose.debug.yml` явно включает HTTP **только на 127.0.0.1:8080**.
На публичном VDS этот файл не подключайте.

Для локального запуска без реального домена нужны OpenSSL и свободный порт 443:

```sh
cp .env.example .env
./scripts/create-local-certificate.sh
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

Чтобы выключить локальный HTTP и удалить его опубликованный порт, пересоздайте
фронтенд с базовой конфигурацией:

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

При запущенном приложении интеграционные проверки:

```sh
BASE_URL=https://example.org python3 tests/smoke.py
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
