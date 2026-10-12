# sample

Веб-приложение для сохранения текстов в PostgreSQL. Структура основана на
[adudkin/prairie](https://github.com/adudkin/prairie): Docker Compose и отдельный
каталог Go-сервиса с Dockerfile. В отличие от референса, Go обслуживает также
HTML-страницы, поэтому всего два контейнера: `web` (Go + страницы) и `db` (PostgreSQL).

## Запуск

Нужны Git, запущенный Docker и Docker Compose v2. Порт 8080 должен быть свободен.

Клонируйте репозиторий и перейдите в каталог проекта:

```sh
git clone --branch work https://github.com/adudkin/sample.git
cd sample
```

Соберите и запустите оба контейнера:

```sh
docker compose up --build -d
```

Дождитесь запуска приложения и проверьте его готовность:

```sh
curl --fail http://localhost:8080/healthz
```

- `http://localhost:8080/` — текстовое поле и кнопка «Сохранить».
- `http://localhost:8080/admin` — все записи, новые сверху.
- `POST /api/texts` — JSON `{"text":"Текст"}`, ответ 201 с сохранённой записью.
- `GET /api/texts` — список записей.
- `GET /healthz` — проверка приложения и соединения с БД.

Таблица создаётся автоматически при запуске. Пустые тексты отклоняются;
максимальный размер текста — 60000 байт. Текст сохраняется без обрезки и
отображается как текст, без исполнения HTML. Страница `/admin` и API открыты
без авторизации; перед публичным размещением добавьте контроль доступа.

PostgreSQL не публикует порт на хост. Для локальной разработки используется
пароль `sample_local_dev`; другой пароль можно задать через `POSTGRES_PASSWORD`
в окружении или в игнорируемом `.env` до первого запуска. При использовании
пароля в DATABASE_URL специальные символы должны быть URL-кодированы.

Данные сохраняются в томе `sample-data` при перезапусках и `docker compose down`.
`docker compose down -v` удаляет данные. Изменение пароля в окружении не меняет
пароль в уже созданной базе.

## Диагностика

Если приложение ещё не отвечает, проверьте состояние контейнеров и журналы:

```sh
docker compose ps
docker compose logs --tail=100 web db
```

После готовности базы повторите запрос `/healthz`.

## Проверка

При запущенном приложении:

```sh
python3 tests/smoke.py
```

Тесты проверяют страницы, готовность БД, сохранение и чтение многострочного
текста и отклонение некорректного ввода. Тест сохранения добавляет одну запись.

Go-код можно собирать Go 1.24+, из каталога `sample-server`:

```sh
go mod download
go build ./...
```

## Остановка

```sh
docker compose down
```

### Сборка в облачном окружении с HTTPS-прокси

Если прокси использует дополнительный доверенный CA, Dockerfile поддерживает
необязательный BuildKit secret `cloud_ca`. Проверка TLS остаётся включённой:

```sh
DOCKER_CONFIG=/tmp/sample-docker-config docker build \
  --add-host "proxy:$(getent ahostsv4 proxy | awk 'NR==1 {print $1}')" \
  --build-arg HTTPS_PROXY --build-arg HTTP_PROXY --build-arg NO_PROXY \
  --secret id=cloud_ca,src=/etc/ssl/certs/ca-certificates.crt \
  -t sample-web ./sample-server
docker compose up -d --no-build
```

Команды выполняются из корня проекта. Сертификат не копируется в итоговый образ.
