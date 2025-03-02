Сервис подсчёта арифметических выражений. Поддерживает операторы +, -, /, *, а также скобочки для приоритезации отдельных 
частей выражения.

# Развёртывание
`git clone https://github.com/Debianov/calc-ya-go-24.git`

В `config.go` в строке `return &http.Server{Addr: "127.0.0.1:8000", Handler: handler}` может быть изменён адрес `Addr`
на любой желаемый.

# Запуск
Запуск: `go run main.go`

# Использование
```shell
curl --location 'localhost:8000/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
  "expression": "2+2*2" 
}'
```
```shell
curl --location 'localhost:8000/api/v1/expressions'
```

```shell
curl --location 'localhost:8000/api/v1/expressions/id'
```

```shell
curl --location 'localhost:8000/internal/task'
```

```shell
curl --location 'localhost:8000/internal/task' \
--header 'Content-Type: application/json' \
--data '{
  "id": 0,
  "result": 2.5
}'
```

Ответ возвращается в формате также в формате json с ключом `"result"`:
```json
{"result":4}
```
```json
{"result":114}
```

В случае некорректного выражения будет возвращён код 422. В случае неизвестной ошибки — 500.

# Участие в разработке
Используйте отдельные ветки и pull request-ы, когда всё готово.