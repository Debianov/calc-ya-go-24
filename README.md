Сервис подсчёта арифметических выражений. Поддерживает операторы +, -, /, *, а также скобочки для приоритезации отдельных 
частей выражения.

# Развёртывание
`git https://github.com/Debianov/calc-ya-go-24.git`

В `config.go` в строке `return &http.Server{Addr: "127.0.0.1:8000", Handler: handler}` может быть изменён адрес `Addr`
на любой желаемый.

TODO Docker?

# Запуск и использование
Запуск: `go run main.go`

Все запросы выполняются в формате json с методом "POST":
```shell
curl --location 'localhost:8000/api/v1/calculate' \
--header 'Content-Type: application/json' \
--data '{
  "expression": "2*2" 
}'
```
```shell
curl --location 'localhost:8000/api/v1/calculate' \ 
--header 'Content-Type: application/json' \
--data '{
  "expression": "2*2+(23+87)"
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