# 🚀 AgentSearch

> **Высокопроизводительный OSINT-инструмент на Go для поиска никнеймов и email по тысячам веб-ресурсов.**

Репозиторий: [github.com/johan-larp/AgentSearch](https://github.com/johan-larp/AgentSearch)

---

## Содержание

- [Обзор](#обзор)
- [Требования](#требования)
- [Установка и сборка](#установка-и-сборка)
- [Быстрый старт](#быстрый-старт)
- [Флаги командной строки](#флаги-командной-строки)
- [База сайтов (конфигурация)](#база-сайтов-конфигурация)
  - [Нативный YAML-формат](#нативный-yaml-формат)
  - [Полный справочник полей SiteConfig](#полный-справочник-полей-siteconfig)
  - [Типы проверок (check_type)](#типы-проверок-check_type)
  - [Совместимость со Sherlock и Maigret](#совместимость-со-sherlock-и-maigret)
- [Движок детекции](#движок-детекции)
  - [Порядок анализа](#порядок-анализа)
  - [Система WAF-детекции](#система-waf-детекции)
  - [Расчёт уверенности (confidence)](#расчёт-уверенности-confidence)
- [Прокси](#прокси)
- [Rate Limiting](#rate-limiting)
- [Вывод результатов](#вывод-результатов)
  - [JSON](#json)
  - [CSV](#csv)
  - [TXT](#txt)
- [Отчёты](#отчёты)
  - [CLI Report](#cli-report)
  - [HTML Report](#html-report)
  - [DOCX Report](#docx-report)
- [Конвертация баз](#конвертация-баз)
- [Graceful Shutdown](#graceful-shutdown)
- [Архитектура](#архитектура)
- [Структура проекта](#структура-проекта)
- [Рецепты и сценарии](#рецепты-и-сценарии)
- [Лицензия](#лицензия)

---

## Обзор

AgentSearch — OSINT-утилита, которая проверяет наличие профиля (или email-адреса) на сотнях и тысячах сайтов одновременно. Основные характеристики:

| Возможность | Описание |
|-------------|----------|
| **Worker Pool** | Фиксированное число горутин-воркеров вместо хаотичного спавна. Контролируемая конкурентность без перегрузки. |
| **Прокси** | Ротация из файла (HTTP/HTTPS/SOCKS5) или программная загрузка с ProxyScrape API. |
| **Smart Detection** | Декларативный движок: статус-коды, подстроки, regex, редиректы, заголовки. |
| **WAF Detection** | Автоматическое распознавание Cloudflare, DDoS-Guard, Incapsula, Sucuri — не считает их за «профиль найден». |
| **uTLS / JA3 Spoofing** | Подделка TLS-отпечатка под реальный Chrome для обхода WAF. |
| **Smart Retry** | Автоматический retry с exponential backoff на 429/5xx ошибках. |
| **Rate Limiting** | Per-host задержка для минимизации 429 и банов. |
| **Structured Logging** | `log/slog` (Go 1.21+) с человекочитаемым выводом в stderr. |
| **Мультиформатный вывод** | JSON (потоковый, без хранения в памяти), CSV, TXT — одновременно. |
| **Rich Reports** | Цветной CLI-вывод, интерактивный HTML и формальный DOCX. |
| **Универсальные базы** | Нативный YAML, Sherlock `data.json`, Maigret `data.json` — без конвертации. |
| **Graceful Shutdown** | Корректная остановка по `Ctrl+C`/`SIGTERM` — воркеры дорабатывают, результаты сохраняются. |
| **Ротация User-Agent** | Встроенный набор + загрузка из файла. |

**Зависимости:**
- `golang.org/x/net` — для SOCKS5-прокси
- `gopkg.in/yaml.v3` — для парсинга YAML-баз
- `github.com/hashicorp/go-retryablehttp` — retry с backoff
- `github.com/refraction-networking/utls` — JA3 fingerprint spoofing
- `github.com/fatih/color` — цветной CLI-вывод
- `github.com/olekukonko/tablewriter` — таблицы в терминале
- `github.com/unidoc/unioffice` — генерация DOCX

---

## Требования

- [Go 1.21+](https://golang.org/dl/)
- Доступ в интернет для проверки сайтов

---

## Установка и сборка

```bash
git clone https://github.com/johan-larp/AgentSearch.git
cd AgentSearch

# Основной бинарник
go build -o agentsearch ./cmd/agentsearch

# Утилита конвертации баз (опционально)
go build -o convert ./cmd/convert
```

После сборки:
- `agentsearch` — основной исполняемый файл
- `convert` — утилита слияния Sherlock + Maigret → YAML
- `configs/sites.yaml` — база сайтов по умолчанию

---

## Быстрый старт

### Поиск одного никнейма

```bash
./agentsearch -u johndoe
```

Результаты сохраняются в `output/`:
```
output/
├── johndoe.json
├── johndoe.csv
├── johndoe.txt
├── johndoe_report.txt
├── johndoe_report.html
└── johndoe_summary.json
```

### Поиск с прокси и увеличенной параллельностью

```bash
./agentsearch -u johndoe -p proxies.txt -w 100 -rl 200ms
```

### Массовый поиск по списку

Создайте `targets.txt`:
```
johndoe
janedoe
admin
```

```bash
./agentsearch -f targets.txt -w 50 -o results -of json,csv
```

### Обход WAF через uTLS + retry

```bash
./agentsearch -u johndoe -s merged_sites.yaml -utls -retries 3 -w 50
```

### Использование Sherlock/Maigret баз напрямую

AgentSearch умеет читать Sherlock `data.json` и Maigret `data.json` **напрямую**, без конвертации:

```bash
# Скачать свежие базы
curl -L -o sherlock_data.json \
  https://raw.githubusercontent.com/sherlock-project/sherlock/master/sherlock_project/resources/data.json

curl -L -o maigret_data.json \
  https://raw.githubusercontent.com/soxoj/maigret/main/maigret/resources/data.json

# Искать сразу по базе Sherlock
./agentsearch -u johndoe -s sherlock_data.json -w 100 -rl 300ms -tt 30m

# Или объединить обе базы в YAML
./convert -o merged_sites.yaml sherlock_data.json maigret_data.json
./agentsearch -u johndoe -s merged_sites.yaml -w 100 -rl 300ms -tt 30m
```

---

## Флаги командной строки

| Флаг | Тип | Описание | По умолчанию |
|------|-----|----------|--------------|
| `-u` | `string` | Целевой никнейм или email | *(обязательно, если нет `-f`)* |
| `-f` | `string` | Путь к файлу со списком целей (одна на строку) | — |
| `-s` | `string` | Путь к базе сайтов (YAML или JSON) | `configs/sites.yaml` |
| `-p` | `string` | Путь к файлу со списком прокси | — |
| `-w` | `int` | Количество конкурентных воркеров | `50` |
| `-rl` | `duration` | Задержка между запросами к одному хосту | `500ms` |
| `-rt` | `duration` | Таймаут одного HTTP-запроса | `15s` |
| `-tt` | `duration` | Общий таймаут на весь поиск (per batch) | `10m` |
| `-o` | `string` | Директория для сохранения результатов | `output` |
| `-of` | `string` | Форматы вывода через запятую: `json`, `csv`, `txt` | `json,csv,txt` |
| `-rf` | `string` | Форматы отчётов: `cli`, `html`, `docx` | `cli,html` |
| `-ua` | `string` | Путь к внешнему файлу со списком User-Agent | — |
| `-mc` | `int` | Максимальное число idle-соединений в пуле | `500` |
| `-mch` | `int` | Максимальное число idle-соединений на хост | `100` |
| `-utls` | `bool` | Включить uTLS JA3 fingerprint spoofing (обход WAF) | `false` |
| `-retries` | `int` | Максимальное число retry при 429/5xx | `2` |
| `-d` | `bool` | Включить Deep Search (dorking mode, stub) | `false` |

> **Примечание:** Должен быть указан хотя бы один из `-u` или `-f`. Если указаны оба — цели суммируются.

### Формат длительности (duration)

Go-duration: `0s`, `100ms`, `5s`, `1m`, `30m`, `1h`. Подробнее: [pkg.go.dev/time#ParseDuration](https://pkg.go.dev/time#ParseDuration)

### Примеры комбинаций

```bash
# Максимально быстрый поиск (агрессивный, много ложных срабатываний)
./agentsearch -u johndoe -w 200 -rl 0s -mc 1000 -mch 200

# Только JSON, тихий режим
./agentsearch -u johndoe -of json -o /tmp/results

# Поиск с кастомной базой и прокси
./agentsearch -u johndoe -s my_sites.yaml -p proxies.txt -w 80 -rl 100ms

# Долгий поиск по объединённой базе (3000+ сайтов)
./agentsearch -u johndoe -s merged_sites.yaml -w 100 -tt 60m -rt 20s

# Обход WAF через uTLS + retry + все форматы отчётов
./agentsearch -u johndoe -s merged_sites.yaml -utls -retries 3 -w 50 -rf cli,html,docx
```

---

## База сайтов (конфигурация)

### Нативный YAML-формат

Файл базы — это **массив объектов** `SiteConfig`. Минимальный пример:

```yaml
- name: GitHub
  url: https://github.com/{username}
  check_type: status_code
  error_code: 404
  weight: 20

- name: Instagram
  url: https://www.instagram.com/{username}/
  check_type: message
  absence_strs:
    - "Sorry, this page isn't available."
  presence_strs:
    - '"biography"'
  weight: 20
```

Переменные в URL:
- `{username}` — подставляется искомый никнейм
- `{target}` — алиас `{username}` (для совместимости)

Эти переменные также подставляются в:
- `request_payload`
- `headers` (значения)
- `presence_strs` / `absence_strs` (подстроки)

### Полный справочник полей SiteConfig

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| `name` | `string` | ✅ | Имя сайта (отображается в логах и отчётах) |
| `url` | `string` | ✅ | URL с плейсхолдером `{username}` |
| `url_probe` | `string` | — | Альтернативный URL для проверки (API endpoint). Если задан — используется вместо `url` |
| `url_main` | `string` | — | Основной URL сайта (для справки, не используется в проверке) |
| `engine` | `string` | — | Движок сайта (информационное поле, не влияет на логику) |
| `check_type` | `string` | — | Тип проверки: `status_code`, `message`, `response_url`, `header` |
| `error_code` | `int` | — | Код ответа, означающий отсутствие профиля (для `status_code`) |
| `error_url` | `string` | — | Паттерн URL, означающий отсутствие (для `response_url`) |
| `absence_strs` | `[]string` | — | Подстроки в теле ответа, означающие **отсутствие** профиля |
| `absence_regexes` | `[]string` | — | Regex-паттерны отсутствия профиля |
| `presence_strs` | `[]string` | — | Подстроки в теле ответа, означающие **наличие** профиля |
| `presence_regexes` | `[]string` | — | Regex-паттерны наличия профиля |
| `follow_redirects` | `bool` | — | Разрешить ли редиректы (по умолчанию `true`). Если `false` — останавливается на первом ответе |
| `redirect_failure_patterns` | `[]string` | — | Паттерны в финальном URL, означающие «профиль не найден» |
| `headers` | `map[string]string` | — | Кастомные HTTP-заголовки запроса |
| `required_headers` | `map[string]string` | — | Заголовки ответа, которые нужно проверить (для `check_type: header`) |
| `request_method` | `string` | — | HTTP-метод (`GET`, `POST`). По умолчанию `GET` |
| `request_payload` | `string` | — | Тело POST-запроса |
| `request_head_only` | `bool` | — | Отправлять HEAD вместо GET (для `status_code`) |
| `regex_check` | `string` | — | Regex для валидации никнейма **до** отправки запроса |
| `weight` | `int` | — | Вес сайта для расчёта confidence (1–100, по умолчанию 10) |
| `tags` | `[]string` | — | Теги сайта (например: `coding`, `ru`, `social`) |
| `disabled` | `bool` | — | Если `true` — сайт исключается из поиска |
| `protection` | `[]string` | — | Список известных защит (`cf_js_challenge`, `custom_bot_protection`) |
| `waf_indicators` | `WAFConfig` | — | Пользовательские правила WAF-детекции (см. ниже) |

#### WAFConfig

```yaml
waf_indicators:
  status_codes: [403]      # Статус-коды, означающие защиту
  header_keys: ["CF-RAY"]  # Наличие заголовков-маркеров
  body_substrings: ["cloudflare"]  # Подстроки в теле ответа
```

### Типы проверок (check_type)

#### `status_code`

Проверяет HTTP-статус ответа. Профиль **не найден**, если код совпадает с `error_code` (обычно `404`).

```yaml
- name: GitHub
  url: https://github.com/{username}
  check_type: status_code
  error_code: 404
```

#### `message`

Анализирует тело ответа на подстроки и regex. Приоритет:

1. Если найдена `absence_strs` / `absence_regexes` → **не найден**
2. Если найдена `presence_strs` / `presence_regexes` → **найден**
3. Если ничего не найдено → **предположительно найден** с пониженной уверенностью (≥30%)

```yaml
- name: Twitter/X
  url: https://x.com/{username}
  check_type: message
  absence_strs:
    - "page doesn't exist"
    - "This account doesn't exist"
  presence_strs:
    - '"screen_name":"{username}"'
```

#### `response_url`

Проверяет финальный URL (после редиректов). Профиль **не найден**, если URL содержит `error_url`.

```yaml
- name: VK_by_id
  url: https://vk.com/id{username}
  check_type: response_url
  error_url: "vk.com/blank.php"
```

> **Важно:** Для `response_url` нужно, чтобы `follow_redirects` был `true` (по умолчанию).

#### `header`

Проверяет наличие определённых заголовков в ответе.

```yaml
- name: SomeSite
  url: https://example.com/api/{username}
  check_type: header
  required_headers:
    X-Profile-Exists: "true"
```

#### Fallback (check_type не указан)

Если тип не задан, движок использует эвристики:
- Если `error_code` задан и совпадает → `not_found`
- Если статус 2xx/3xx → `found` с базовой уверенностью
- Иначе → `not_found`

### Совместимость со Sherlock и Maigret

AgentSearch принимает **три формата** баз данных без предварительной конвертации:

| Формат | Структура | Пример |
|--------|-----------|--------|
| **Нативный YAML** | `[]SiteConfig` — прямой массив | `configs/sites.yaml` |
| **Sherlock JSON** | `{ "SiteName": { "url": "...", "errorType": "..." } }` | `sherlock/data.json` |
| **Maigret JSON** | `{ "sites": { "SiteName": { ... } } }` | `maigret/data.json` |

Логика загрузки (`internal/config/loader.go`):
1. Пробуем распарсить как прямой YAML-массив
2. Пробуем как JSON/YAML объект с полем `sites` (map)
3. Пробуем как JSON/YAML объект с полем `sites` (array)
4. Пробуем как плоский Sherlock JSON (ключ = имя сайта)

При парсинге Maigret/Sherlock:
- Автоматически определяется `check_type` на основе полей (`error_code` → `status_code`, `absence_strs` → `message`, `error_url` → `response_url` и т.д.)
- `weight` по умолчанию устанавливается в `10`
- Записи с `disabled: true` пропускаются
- Записи без URL (engine-only) игнорируются (пустой URL → пропуск в website source)

---

## Движок детекции

Движок (`internal/detector/detector.go`) — сердце AgentSearch. Он анализирует HTTP-ответ по декларативным правилам из `SiteConfig`.

### Порядок анализа

```
1. WAF / защита
   ├── Глобальные эвристики (CF-RAY, Server: cloudflare, 429, ddos-guard, incapsula, sucuri)
   ├── Пользовательские waf_indicators (status_codes, header_keys, body_substrings)
   └── Поле protection (cf_js_challenge, custom_bot_protection)
   → Если WAF обнаружен: status = "blocked", found = false

2. Редиректы
   └── redirect_failure_patterns: если финальный URL содержит паттерн → "not_found"

3. Декларативная проверка (по check_type)
   ├── status_code
   ├── message
   ├── response_url
   └── header

4. Fallback (если check_type не задан)
```

### Система WAF-детекции

WAF детектируется **до** основной проверки, чтобы не засчитать заглушку защиты за реальный профиль.

**Глобальные эвристики** (работают для всех сайтов автоматически):

| Сигнал | Условие |
|--------|---------|
| Cloudflare header | `CF-RAY` присутствует в заголовках |
| Cloudflare server | Заголовок `Server` содержит `cloudflare` |
| Cloudflare challenge | Статус `403` + тело содержит `cloudflare` или `ray id` |
| Rate limit | Статус `429` |
| DDoS-Guard | Тело содержит `ddos-guard` |
| Incapsula | Тело содержит `incapsula` |
| Sucuri | Тело содержит `sucuri` |

**Пользовательские правила** (из `waf_indicators` конкретной записи):

```yaml
waf_indicators:
  status_codes: [403]              # Любые 403 от этого сайта → считаем WAF
  header_keys: ["CF-RAY"]          # Наличие заголовка → WAF
  body_substrings: ["cloudflare", "cf-error"]  # Подстроки в теле → WAF
```

### Расчёт уверенности (confidence)

Confidence — число от 0 до 100, отражающее «надёжность» результата.

Алгоритм (`calculateBaseConfidence`):

| Фактор | Баллы |
|--------|-------|
| `weight` сайта | По умолчанию 10, макс. зависит от конфигурации |
| HTTP 200 | +30 |
| Тело > 200 байт | +10 |
| **Кап** | 100 |

Для `check_type: message` без явных индикаторов: минимальная уверенность = 30%.

**Интерпретация:**
- **0%** — профиль не найден или заблокирован WAF
- **1–30%** — низкая уверенность, возможные ложные срабатывания
- **31–60%** — умеренная уверенность
- **61–100%** — высокая уверенность, профиль скорее всего существует

---

## Прокси

### Формат файла

Создайте `proxies.txt`:

```
# Комментарии начинаются с #
http://123.45.67.89:8080
https://98.76.54.32:443
socks5://192.168.1.1:1080
socks5h://proxy.example.com:1080

# Если схема не указана — подразумевается http://
11.22.33.44:3128

# Прокси с авторизацией
http://user:pass@proxy.example.com:3128
```

### Поддерживаемые протоколы

| Протокол | Описание |
|----------|----------|
| `http://` | HTTP-прокси |
| `https://` | HTTP-прокси через TLS |
| `socks5://` | SOCKS5 с локальным DNS-резолвом |
| `socks5h://` | SOCKS5 с DNS-резолвом на стороне прокси |

### Ротация

Прокси ротируются по **кольцевому алгоритму** (round-robin). Каждый запрос получает следующий прокси в списке. Реализация потокобезопасна (`atomic.Uint64`).

### Режимы работы HTTP-клиента

- **Без прокси** — единый `http.Transport` с агрессивным пулом соединений (быстрее)
- **С прокси** — транспорт клонируется под каждый запрос для изоляции настроек прокси (избегаем data race). Соединения не переиспользуются между разными прокси.

### Автозагрузка с ProxyScrape API

Программно (из Go-кода):

```go
import "github.com/johan-larp/agentsearch/internal/network"

proxies := network.FetchProxies()
// proxies == []string{"1.2.3.4:8080", "5.6.7.8:1080", ...}
```

API: `https://api.proxyscrape.com/v4/free-proxy-list/get?request=display_proxies&proxy_format=protocolipport&format=text`

---

## Rate Limiting

`HostLimiter` (`internal/ratelimit/limiter.go`) обеспечивает задержку между запросами к **одному и тому же хосту**.

**Принцип работы:**
- Хранит `map[host]time.Time` — время последнего запроса к каждому хосту
- Перед запросом проверяет: если с последнего запроса прошло меньше `rl` — ждёт разницу
- Потокобезопасен (`sync.Mutex`)

```
HostLimiter.Wait("https://github.com/johndoe")
 → извлекает host = "github.com"
 → если последний запрос к github.com был < 500ms назад → sleep(500ms - elapsed)
```

**Рекомендации:**
- Без прокси: `-rl 500ms` или выше
- С прокси: `-rl 100ms-200ms` (т.к. запросы идут с разных IP)
- `-rl 0s` — отключает лимит (для агрессивного поиска)

---

## Вывод результатов

Все три формата пишутся **одновременно** и **потоково** (результат не хранится в памяти целиком).

Имя файла файла формируется из цели: `output/{target}.{format}`.

Недопустимые символы в имени (`/ : * ? " < > |`) заменяются на `_`.

### JSON

Потоковая запись массива. Файл открывается `[`, каждый результат дописывается через запятую, в конце `]`.

```json
[
  {"site_name":"GitHub","target":"johndoe","url":"https://github.com/johndoe","found":false,"confidence":0,"status":"not_found","duration":56200000,"error":"","final_url":""},
  {"site_name":"Twitter/X","target":"johndoe","url":"https://x.com/johndoe","found":false,"confidence":0,"status":"blocked","duration":187000000,"error":"","final_url":""},
  {"site_name":"Instagram","target":"johndoe","url":"https://www.instagram.com/johndoe/","found":true,"confidence":30,"status":"found","duration":164000000,"error":"","final_url":""}
]
```

**Поля JSON:**

| Поле | Тип | Описание |
|------|-----|----------|
| `site_name` | `string` | Имя сайта |
| `target` | `string` | Искомый никнейм/email |
| `url` | `string` | URL, по которому выполнялся запрос |
| `found` | `bool` | Найден ли профиль |
| `confidence` | `int` | Уверенность (0–100) |
| `status` | `string` | Статус: `found`, `not_found`, `blocked`, `error` |
| `duration` | `int64` | Время запроса в наносекундах |
| `error` | `string` | Текст ошибки (если статус `error`) |
| `final_url` | `string` | Финальный URL после редиректов (если отличается от `url`) |

### CSV

Стандартный CSV с заголовком. `duration` — в миллисекундах.

```csv
site_name,target,url,found,confidence,status,duration_ms,error,final_url
GitHub,johndoe,https://github.com/johndoe,false,0,not_found,56,,
Twitter/X,johndoe,https://x.com/johndoe,false,0,blocked,187,,
Instagram,johndoe,https://www.instagram.com/johndoe/,true,30,found,164,,
```

### TXT

Человекочитаемый формат, одна строка на результат:

```
[GitHub] johndoe | Found: false | Confidence: 0% | Status: not_found | URL: https://github.com/johndoe
[Twitter/X] johndoe | Found: false | Confidence: 0% | Status: blocked | URL: https://x.com/johndoe
[Instagram] johndoe | Found: true | Confidence: 30% | Status: found | URL: https://www.instagram.com/johndoe/
  -> Final URL: https://www.instagram.com/some_redirect/
```

---

## Отчёты

AgentSearch генерирует отчёты после каждого поиска (флаг `-rf`). Отчёты строятся на основе **всех** результатов (found, blocked, error) и содержат агрегированную статистику.

### CLI Report (`*_report.txt`)

Цветной вывод в терминал с таблицей найденных профилей, статистикой и списком заблокированных сайтов:

```
╔══════════════════════════════════════════════════════════════╗
║           AGENTSEARCH OSINT REPORT                           ║
╚══════════════════════════════════════════════════════════════╝
Target:    johndoe
Duration:  2m15s

Statistics:
  Total checks:  2063
  Found:         342 (16.6%)
  Not Found:     1201 (58.2%)
  Blocked:       412 (20.0%)
  Errors:        108 (5.2%)

✓ Found 342 profile(s):

SITE                           CONFIDENCE   LATENCY      URL
────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
GitHub                         70%          245ms        https://github.com/johndoe
Twitter/X                      45%          189ms        https://x.com/johndoe
...
```

Также сохраняется в `output/*_report.txt`.

### HTML Report (`*_report.html`)

Интерактивный отчёт в тёмной теме GitHub-style с:
- Карточками статистики (Total / Found / Blocked / Errors)
- Фильтруемой таблицей найденных профилей (JavaScript `filterTable`)
- Визуальными confidence-bar
- Секциями Blocked by WAF и Errors

Открывается в браузере — можно сортировать, фильтровать и печатать в PDF.

### DOCX Report (`*_report.docx`)

Формальный Word-документ для передачи заказчику:
- Заголовок и мета-информация
- Сводка в bullet-списке
- Найденные профили с confidence
- Список WAF-блокировок
- Ошибки запросов

### Summary JSON (`*_summary.json`)

Автоматически генерируется рядом с отчётами:

```json
{
  "target": "johndoe",
  "total": 2063,
  "found": 342,
  "not_found": 1201,
  "blocked": 412,
  "errors": 108,
  "duration": "2m15s"
}
```

---

## Конвертация баз

### Вариант 1: Использовать базы напрямую

AgentSearch читает Sherlock `data.json` и Maigret `data.json` без конвертации:

```bash
./agentsearch -u johndoe -s sherlock_data.json
./agentsearch -u johndoe -s maigret_data.json
```

### Вариант 2: Объединить через утилиту `convert`

```bash
# Скачать базы
curl -L -o sherlock_data.json \
  https://raw.githubusercontent.com/sherlock-project/sherlock/master/sherlock_project/resources/data.json

curl -L -o maigret_data.json \
  https://raw.githubusercontent.com/soxoj/maigret/main/maigret/resources/data.json

# Объединить в единый YAML
./convert -o merged_sites.yaml sherlock_data.json maigret_data.json

# Искать
./agentsearch -u johndoe -s merged_sites.yaml -w 100 -rl 300ms
```

**Что делает конвертер:**
- Заменяет `{}` → `{username}` (Sherlock)
- Нормализует `errorMsg` (строка или массив) → `absence_strs`
- Сливает `presenceStrs` + `presenseStrs` (Maigret опечатка)
- Дедуплицирует по имени+URL
- Отбрасывает записи без URL (engine-only сайты)

### Вариант 3: Две базы — два запуска

```bash
./agentsearch -u johndoe -s sherlock_data.json -o results/sherlock
./agentsearch -u johndoe -s maigret_data.json -o results/maigret
```

---

## Graceful Shutdown

При получении `SIGINT` (Ctrl+C) или `SIGTERM`:

1. Контекст отменяется (`signal.NotifyContext`)
2. Новые задачи не подаются в пул
3. Воркеры дорабатывают текущие запросы
4. Результаты, уже полученные, записываются в файлы (writer'ы флашат буферы)
5. Отчёты генерируются на основе собранных результатов
6. Программа завершается с кодом 0

```
$ ./agentsearch -u johndoe -s merged_sites.yaml -w 100
time=2026-07-09T17:19:40.000Z level=INFO msg="starting search" target=johndoe workers=100
time=2026-07-09T17:19:41.000Z level=INFO msg="profile found" site=GitHub url=...
^C
time=2026-07-09T17:19:45.000Z level=WARN msg="shutdown signal received, exiting"
```

Результаты, полученные до `Ctrl+C`, **не теряются**.

---

## Архитектура

```text
CLI (-u / -f, прежние флаги)
  ↓
Application: зависимости и жизненный цикл → Runner: типизированный поиск
  ↓
Sources: реестр и диспетчеризация по возможностям
  └─ websites → worker pool → rate limiter → network → detector / WAF
  ↓
Нормализованные results
  ↓
Общие storage / reports
```

Поиск по сайтам теперь реализован как **один источник** —
`internal/sources/websites`. Его процессор перенесён из `app.go`; HTTP-клиент,
детектор, настройки сайтов и пул воркеров переиспользуются, а не дублируются.
Имя источника в реестре — `websites`, поле `source` в результате — имя конкретного
сайта из базы. Другой провайдер может выполнять локальные операции без HTTP и воркеров.

`Source` предоставляет имя и тип. Возможности определяются отдельными интерфейсами:
`UsernameSearcher`, `EmailSearcher`, `PasswordSearcher`, `PasswordHashSearcher`.
Реестр сохраняет порядок регистрации; диспетчер вызывает только поддерживаемые
операции. Результаты передаются последовательным callback с backpressure.
`app.NewRunner(registry).Search(ctx, target, emit)` не требует CLI или файлового вывода.

Модель `Target` поддерживает `username`, `email`, `password`, `password_hash`.
Пароли и хеши скрыты при форматировании/сериализации; в результат переносится только
безопасная ссылка (`target` + `target_type`). Пакет `security` редактирует известные
секреты, чувствительные поля и учётные данные в URL. Это дополнительная защита:
провайдеры не должны помещать секреты в evidence, metadata, URL или ошибки.
CLI по-прежнему принимает только прежние `-u` / `-f`; **не передавайте туда пароли**.

JSON сохраняет прежние поля и добавляет `source`, `source_type`, `target_type`,
а при наличии — `evidence` и `metadata`. CSV сохраняет прежние девять колонок,
TXT — прежний краткий формат: это совместимые проекции общего результата.
HTML умеет отображать дополнительные evidence/metadata без зависимости от провайдера.
Storage и reports не импортируют конкретные sources; network не зависит от sources.
Models зависят только от стандартной библиотеки и небольшого пакета security.

`configs/sites.yaml` и все CLI-флаги сохранены. `configs/services.yaml` —
**неактивный пример** будущих настроек: `config.LoadServices` только читает и проверяет
YAML, не читает ключи из окружения и не запускает сервисы. CLI этот файл не загружает.
HIBP, Pwned Passwords, локальная база паролей, HTTP API и AI **не реализованы**.

---

## Структура проекта

```
AgentSearch/
├── cmd/
│   ├── agentsearch/
│   │   └── main.go              # Точка входа: логирование, парсинг флагов, запуск App
│   └── convert/
│       └── main.go              # Утилита слияния Sherlock + Maigret → YAML
├── configs/
│   ├── sites.yaml               # База сайтов по умолчанию (примеры)
│   └── services.yaml            # Неактивные настройки будущих сервисов
├── internal/
│   ├── app/
│   │   ├── app.go               # Создание зависимостей и регистрация sources
│   │   └── runner.go            # Типизированный поиск и CLI lifecycle
│   ├── config/
│   │   ├── config.go            # ParseFlags(), AppConfig
│   │   ├── loader.go            # LoadSites() — загрузка YAML/JSON/Sherlock/Maigret
│   │   ├── convert.go           # MergeAndConvert() — логика слияния баз
│   │   └── services.go          # Независимая загрузка настроек сервисов
│   ├── detector/
│   │   └── detector.go          # Engine, Analyze(), WAF-детекция, confidence
│   ├── models/
│   │   ├── site.go              # SiteConfig, WAFConfig
│   │   ├── target.go            # Target, TargetType
│   │   ├── result.go            # Общий Result и совместимые поля
│   │   └── source.go            # SourceType, Evidence
│   ├── security/
│   │   └── redact.go            # Secret и редактирование чувствительных данных
│   ├── sources/
│   │   ├── source.go            # Интерфейсы возможностей
│   │   ├── registry.go          # Регистрация и поиск источников
│   │   ├── manager.go           # Диспетчеризация и нормализация
│   │   └── websites/
│   │       ├── source.go        # Адаптер существующего движка
│   │       └── processor.go     # Обработка сайта через network и detector
│   ├── network/
│   │   ├── client.go            # HTTP-клиент, proxyRotatorTransport, retryable client
│   │   ├── utls.go              # uTLS JA3 fingerprint spoofing
│   │   ├── retry.go             # go-retryablehttp wrapper
│   │   ├── proxy.go             # ProxyRotator, FetchProxies()
│   │   └── ua.go                # UARotator (встроенный + из файла)
│   ├── ratelimit/
│   │   └── limiter.go           # HostLimiter (per-host delay)
│   ├── report/
│   │   ├── report.go            # BuildSummary(), WriteAll()
│   │   ├── cli.go               # Цветной CLI-отчёт
│   │   ├── html.go              # Интерактивный HTML-отчёт
│   │   └── docx.go              # Word/DOCX-отчёт
│   ├── storage/
│   │   ├── manager.go           # Manager — мультиформатная запись
│   │   ├── json.go              # JSONWriter (потоковый)
│   │   ├── csv.go               # CSVWriter
│   │   └── txt.go               # TXTWriter
│   └── worker/
│       └── pool.go              # Pool, Job, Processor interface
├── go.mod
├── go.sum
├── LICENSE
├── proxies.txt                  # Пример/заготовка для прокси
└── README.md
```

---

## Рецепты и сценарии

### Сценарий 1: Быстрый единичный поиск

```bash
./agentsearch -u johndoe -w 100 -rl 100ms
```

50–100 воркеров, 100ms rate limit. Результат за ~10–30 секунд по ~500 сайтам.

### Сценарий 2: Точный поиск с минимизацией ложных срабатываний

```bash
./agentsearch -u johndoe -w 30 -rl 1000ms -rt 20s -tt 30m
```

Меньше воркеров, большая задержка между запросами, увеличенные таймауты.

### Сценарий 3: Массовый поиск по email-адресам

Создайте `emails.txt`:
```
user1@gmail.com
user2@protonmail.com
```

```bash
./agentsearch -f emails.txt -w 50 -o email_results -of json,csv
```

### Сценарий 4: Поиск с прокси и обходом WAF

```bash
./agentsearch -u johndoe -p proxies.txt -w 80 -rl 150ms -utls -retries 3
```

### Сценарий 5: Добавление собственного сайта

Добавьте в `configs/sites.yaml`:

```yaml
- name: MyCustomForum
  url: https://forum.example.com/profile/{username}
  check_type: message
  absence_strs:
    - "User not found"
    - "Profile doesn't exist"
  presence_strs:
    - "Member since"
  weight: 15
  tags:
    - forum
```

### Сценарий 6: Поиск на сайте с POST-запросом

```yaml
- name: APISite
  url: https://api.example.com/check
  request_method: POST
  request_payload: '{"username":"{username}"}'
  headers:
    Content-Type: "application/json"
    Authorization: "Bearer token123"
  check_type: message
  presence_strs:
    - '"exists":true'
  weight: 20
```

### Сценарий 7: Только JSON-вывод в конкретную директорию

```bash
./agentsearch -u johndoe -of json -o /tmp/osint_results
```

### Сценарий 8: Полный набор отчётов для заказчика

```bash
./agentsearch -u johndoe -s merged_sites.yaml -w 100 -rf cli,html,docx -o client_report
```

### Тонкая настройка производительности

| Параметр | Эффект | Рекомендация |
|----------|--------|--------------|
| `-w 20` | Мало воркеров, медленно, но безопасно | Для чувствительных целей |
| `-w 200` | Много воркеров, быстро, но риск банов | С прокси |
| `-rl 0s` | Без задержек | Максимальная скорость, много 429 |
| `-rl 2s` | 2 секунды между запросами к хосту | Мягкий режим |
| `-mc 1000` | Большой пул соединений | Для `-w 200+` |
| `-mch 200` | Больше соединений на хост | Если много запросов к одному домену |
| `-rt 30s` | Долгий таймаут запроса | Для медленных сайтов |
| `-tt 60m` | Долгий общий таймаут | Для 3000+ сайтов |
| `-utls` | JA3 spoofing | Обход Cloudflare TLS fingerprinting |
| `-retries 3` | 3 retry при ошибках | Для нестабильных прокси/сайтов |

---

## Лицензия

MIT License. См. [LICENSE](https://github.com/johan-larp/AgentSearch/blob/main/LICENSE).

---

> **Дисклеймер:** Данный инструмент предназначен исключительно для образовательных и исследовательских целей. Пользователь несёт ответственность за соблюдение законодательства и правил использования веб-сервисов при применении AgentSearch.
