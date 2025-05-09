Сервис подсчёта арифметических выражений. Поддерживает операторы +, -, /, *, а также скобочки для приоритизации \
отдельных частей выражения.

Проект разделён на оркестратор и агент. Оркестратор отвечает за приём новых выражений и регистрацию пользователей,
а агент — за математические вычисления.

# Развёртывание
`git clone https://github.com/Debianov/calc-ya-go-24.git`

## Конфиг-файлы

Конфигурирование оркестратора и агента может быть осуществлён в `backend/orchestrator/config.go` и 
`backend/agent/config.go`

Для работы программы желательна последняя версия Go 1.24 ([как обновить Go](https://go.dev/doc/install), 
если в репозиториях пакетных менеджеров ещё нет новой версии). **Работа проекта протестирована на 
версии 1.24.**

## Переменные среды
Для изменения стандартных настроек оркестратора можно использовать следующие переменные среды:
```
TIME_ADDITION
TIME_SUBTRACTION
TIME_MULTIPLICATIONS
TIME_DIVISIONS
```
Формат значений переменных: `<число><ns/us/ms/s/m>`

Переменные среды для агента:
```
COMPUTING_POWER
```
Формат значений: число.


Пример файла переменных в Linux:
```shell
# filename: calc.env
#!/bin/sh
export TIME_ADDITION=2s
export TIME_SUBTRACTION=2s
export TIME_MULTIPLICATIONS=2s
export TIME_DIVISIONS=2s
export COMPUTING_POWER=10
```

Экспортирование переменных в Linux:
`source calc.env`

# Запуск

Запуск оркестратора:
```shell
cd ./backend/orchestrator
go run github.com/Debianov/calc-ya-go-24/backend/orchestrator
```
Запуск агента:
```shell
cd ./backend/agent
go run github.com/Debianov/calc-ya-go-24/backend/agent
```
Для успешного запуска агента необходимо, чтобы оркестратор был запущен.

# Использование


## Получение токена
Для использования любых доступных endpoint-ов необходимо наличие токена. Токен выдаётся через `/api/v1/login`.
Требуется регистрация через `/api/v1/register`:
```shell
curl -v --location 'localhost:8000/api/v1/register' \
--header 'Content-Type: application/json' \ 
--data '{"login": "test", "password": "qwerty"}'
```

Получить токен:
```shell
curl -v --location 'localhost:8000/api/v1/login' \
--header 'Content-Type: application/json' \ 
--data '{"login": "test", "password": "qwerty"}'
```
Вывод:
```shell
{"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ..."}
```

## Подсчёт и выдача результатов
Запрос на регистрацию нового выражения:
```shell
curl --location 'localhost:8000/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
  "token": "<вставитьТокен>",
  "expression": "2+2*4" 
}'
```
Вывод при статусе 201:
```shell
{"id":<int>}
```

Запрос на получение списка выражений:
```shell
curl --location 'localhost:8000/api/v1/expressions' \
--header 'Content-Type: application/json' \
--data '{
  "token": "<вставитьТокен>" 
}'
```
Вывод при статусе 200:
```shell
{"expressions":[{"id":5,"status":"Выполнено","result":4},{"id":6,"status":"Выполнено","result":4}]}
```

Запрос на получение конкретного выражения по id:
```shell
curl --location 'localhost:8000/api/v1/expressions/<int>' \
--header 'Content-Type: application/json' \
--data '{
  "token": "<вставитьТокен>" 
}'
```
Вывод при статусе 200:
```shell
{"expression":{"id":5,"status":"Выполнено","result":4}}
```

# Участие в разработке

## Pull Request-ы
Используйте отдельные ветки и pull request-ы, когда всё готово.

## Тестирование
Для работы также необходимы экспортированные переменные окружения.

Тесты для оркестратора:
```shell
cd ./backend/orchestrator
go test
```
Тесты для агента:
```shell
cd ./backend/agent
go test
```
Интеграционные тесты:
```shell
cd ./backend
go test
```