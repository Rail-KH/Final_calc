# Распределённый вычислитель арифметических выражений

Этот проект реализует веб-сервис, принимающий арифметическое выражение через HTTP запрос и возвращающий результат вычислений. Результаты вычислений отправляются в базу данных.

- Поддерживаемые операции: `+`, `-`, `*`, `/`
- Приоритет операций и скобки
- Параллельное выполнение операций
- Хранение результатов в базе данных
- Реализована персистентность
  
# Структура проекта
### cmd/ — точка входа приложения
### internal/ — внутренняя логика и модули приложения
### pkg/ — вспомогательные пакеты
### tests_module/ — модульные тесты
### tests_integration/ — интеграционные тесты

# Архитектура

#### Сервис состоит из двух серверов: Оркестратора и Агента

- Оркестратор принимает выражения через HTTP запрос, разбивает выражения на подзадачи и хранит статусы вычислений в базе данных

- Агент запускает несколько горутин, которые получают задачи через HTTP-запросы от Оркестратора, выполняют арифметические операции, возвращают результаты Оркестратору

```mermaid
graph TD
    U[Пользователь] -->|POST /calculate| O[Оркестратор]
    U -->|GET /expressions| O
    O -->|GET /internal/task| A1[Агент Горутина 1]
    O -->|GET /internal/task| A2[Агент Горутина 2]
    A1 -->|POST /internal/task| O
    A2 -->|POST /internal/task| O
```

# Инструкция по запуску:

#### Убедитесь, что у вас установлен Go (желательно версия 1.23.5 или выше).

#### Установить gcc компилятор (так как Sqlite использует язык C, для которого нужен этот компилятор) https://sourceforge.net/projects/tdm-gcc/files/TDM-GCC%20Installer/

## Скопируйте репозиторий:

~~~powershell
git clone https://github.com/Rail-KH/Final_calc
~~~

~~~powershell
cd Final_calc
~~~

## Запустите Оркестратор:

~~~powershell
# Установка времени операций (в миллисекундах)
$env:TIME_ADDITION_MS = "20"
$env:TIME_SUBTRACTION_MS = "20"
$env:TIME_MULTIPLICATIONS_MS = "20"
$env:TIME_DIVISIONS_MS = "20"

# Запуск оркестратора
go run .\cmd\orchestrator\main.go
~~~

## Запустите Агента:

~~~powershell
# Указание вычислительной мощности (количество горутин) и URL оркестратора
$env:COMPUTING_POWER = "4"
$env:ORCHESTRATOR_URL = "http://localhost:8080"

# Запуск агента
go run .\cmd\agent\main.go
~~~

# API Endpoints:

### 1. Регистрация пользователя

```bash
POST /api/v1/register
```

Пример запроса:

```bash
curl --location 'http://localhost:8080/api/v1/register' \
--header 'Content-Type: application/json' \
--data '{
  "login": "Rail",
  "password": "1234"
}'
```


Успешный ответ (200):

```json
{
    "id": 1,
    "login": "Rail"
}
```

### 2. Вход пользователя

```bash
POST /api/v1/login
```

Пример запроса:

```bash
curl --location 'http://localhost:8080/api/v1/login' \
--header 'Content-Type: application/json' \
--data '{
  "login": "Rail",
  "password": "1234"
}'
```


Успешный ответ (200):

```json
{
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySUQiOjF9.ZW7sZI3lwPv9HtnlapjZMk2kAmr-Sy_BjwMe2x4xVDg",
    "user": {
        "id": 1,
        "login": "Rail"
    }
}
```
В ответе возращается JWT-token. Его необходимо будет указывать при следующих запросах


### 3. Добавление выражения

```bash
POST /api/v1/calculate
```

Пример запроса:

```bash
curl --location 'http://localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySUQiOjF9.ZW7sZI3lwPv9HtnlapjZMk2kAmr-Sy_BjwMe2x4xVDg' \
--data '{
  "expression": "(2+3)*4-10/2"
}'
```

Успешный ответ (201):

```json
{
  "id": "1"
}
```

### 4. Получение списка выражений

```bash
GET /api/v1/expressions
```

Пример запроса:

```bash
curl --location 'localhost:8080/api/v1/expressions' \
--header 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySUQiOjF9.ZW7sZI3lwPv9HtnlapjZMk2kAmr-Sy_BjwMe2x4xVDg'
```

Пример ответа (200):

```json
{
    "expressions": [
        {
            "id": "1",
            "expression": "(2+3)*4-10/2",
            "status": "completed",
            "result": 15
        },
    ]
}
```

### 5. Получение выражения по ID

```bash
GET /api/v1/expressions/:id
```

Пример запроса:

```bash
curl --location 'localhost:8080/api/v1/expressions/:1' \
--header 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySUQiOjF9.ZW7sZI3lwPv9HtnlapjZMk2kAmr-Sy_BjwMe2x4xVDg'
```

Ответ (200):

```json
{
    "expression": {
        "id": "1",
        "expression": "(2+3)*4-10/2",
        "status": "completed",
        "result": 15
    }
}
```

# Внутреннее API (для взаимодействия горутин Агента с Оркестратором)

### 1. Получение задачи

```bash
GET /internal/task
```

Пример ответа (200):

```json
{
    "task": {
        "id": "5",
        "arg1": 2,
        "arg2": 3,
        "operation": "+",
        "operation_time": 200
    }
}
```

### 2. Отправка результата

```bash
POST /internal/task
```

Пример запроса:

```json
{
  "id": "5",
  "result": 5
}
```

# Переменные окружения

### Оркестратор

- `PORT` - порт сервера (по умолчанию 8080)
- `TIME_ADDITION_MS` - время сложения (мс)
- `TIME_SUBTRACTION_MS` - время вычитания (мс)
- `TIME_MULTIPLICATIONS_MS` - время умножения (мс)
- `TIME_DIVISIONS_MS` - время деления (мс)

### Агент

- `ORCHESTRATOR_URL` - URL оркестратора
- `COMPUTING_POWER` - количество горутин

# Примеры сценариев

### Сценарий 1: Успешное вычисление

```bash
# Отправка выражения
curl --location 'http://localhost:8080/api/v1/calculate' \
--header 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySUQiOjF9.ZW7sZI3lwPv9HtnlapjZMk2kAmr-Sy_BjwMe2x4xVDg' \
--header 'Content-Type: application/json' \
--data '{
  "expression": "(2+3)*4-10/2"
}'

# Проверка статуса
curl --location 'localhost:8080/api/v1/expressions/:1' \
--header 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySUQiOjF9.ZW7sZI3lwPv9HtnlapjZMk2kAmr-Sy_BjwMe2x4xVDg'

# Ответ(статус 200):
{
    "expression": {
        "id": "1",
        "expression": "(2+3)*4-10/2",
        "status": "completed",
        "result": 15
    }
}
```

### Сценарий 2: Ошибка деления на ноль

```bash
curl --location 'http://localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySUQiOjF9.ZW7sZI3lwPv9HtnlapjZMk2kAmr-Sy_BjwMe2x4xVDg' \
--data '{
  "expression": "10/(5-5)"
}'

# Проверка статуса
curl --location 'localhost:8080/api/v1/expressions/:2' \
--header 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySUQiOjF9.ZW7sZI3lwPv9HtnlapjZMk2kAmr-Sy_BjwMe2x4xVDg'

# Ответ(статус 200):
{
    "expression": {
        "id": "3",
        "expression": "10/(5-5)",
        "status": "completed",
        "result": null
    }
}
```

### Сценарий 3: Невалидные данные

```bash
curl --location 'http://localhost:8080/api/v1/calculate' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VySUQiOjF9.ZW7sZI3lwPv9HtnlapjZMk2kAmr-Sy_BjwMe2x4xVDg' \
--data '{
  "expression":"
}'

# Ответ(статус 422):
{"error":"Invalid Body"}

```
# Тесты

#### Интеграционные тесты

~~~powershell
cd Final_calc
go test ./tests_integration

# ok      github.com/Rail-KH/Final_calc/tests_integration 0.910s
~~~

#### Модульные тесты

~~~powershell
cd Final_calc
go test ./tests_module

# ok      github.com/Rail-KH/Final_calc/tests_module      0.265s
~~~

### Для запросов можно использовать программу Postman
