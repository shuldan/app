# `app` — Гибкий и минималистичный фреймворк для запуска Go-приложений

[![Go CI](https://github.com/shuldan/app/workflows/Go%20CI/badge.svg)](https://github.com/shuldan/app/actions)
[![codecov](https://codecov.io/gh/shuldan/app/branch/main/graph/badge.svg)](https://codecov.io/gh/shuldan/app)
[![Go Report Card](https://goreportcard.com/badge/github.com/shuldan/app)](https://goreportcard.com/report/github.com/shuldan/app)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

Этот проект предоставляет минималистичный, но мощный каркас для построения отказоустойчивых Go-приложений с поддержкой модульной архитектуры, graceful shutdown и управлением жизненным циклом компонентов.

---

## 🚀 Основные возможности

- **Модульная архитектура**: подключайте компоненты как реализации интерфейса `Module`.
- **Graceful shutdown**: корректное завершение работы с таймаутом.
- **Информативные логи**: автоматическая запись событий запуска и остановки.
- **Идемпотентность**: повторный запуск или остановка приложения не вызывает ошибок.
- **Поддержка сигналов ОС**: обработка `SIGINT` и `SIGTERM`.
- **Тестирование**: полный набор unit-тестов с покрытием всех сценариев.

---

## 📦 Установка зависимостей и инструментов

Для работы с проектом убедитесь, что у вас установлен Go 1.24+.

Установите необходимые инструменты:

```sh
make install-tools
```

Это установит:
- `golangci-lint` (v2.4.0)
- `goimports`
- `gosec`

---

## 🛠️ Работа с проектом

### Запуск локальной проверки

```sh
make all
```

Выполняет:
- проверку форматирования кода,
- статический анализ (`golangci-lint`),
- security-сканирование (`gosec`),
- запуск тестов.

### Проверка в CI

```sh
make ci
```

Запускает:
- форматирование,
- `go vet`,
- линтер,
- тесты с отчётом о покрытии.

### Форматирование кода

```sh
make fmt
```

Автоматически форматирует `.go` файлы и сортирует импорты.

### Запуск тестов

```sh
make test
make test-coverage
```

---

## 🧱 Архитектура

### `Application`

Основной объект приложения. Поддерживает опции:

- `WithName(name string)`
- `WithVersion(version string)`
- `WithEnvironment(env string)`
- `WithGracefulTimeout(duration time.Duration)`

### `Module`

Любой компонент должен реализовывать интерфейс:

```go
type Module interface {
	Register(ctx context.Context) error // Выполняется до Start
	Start(ctx context.Context) error    // Запуск компонента
	Stop(ctx context.Context) error     // Корректное завершение
}
```

Модули регистрируются в порядке добавления и останавливаются в обратном порядке.

---

## 🧪 Пример использования

```go
package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/shuldan/app/pkg/application"
)

type HTTPServer struct{}

func (h *HTTPServer) Register(_ context.Context) error {
	slog.Info("HTTP server registering...")
	return nil
}

func (h *HTTPServer) Start(_ context.Context) error {
	slog.Info("HTTP server starting...")
	// запуск сервера
	return nil
}

func (h *HTTPServer) Stop(_ context.Context) error {
	slog.Info("HTTP server stopping...")
	// graceful shutdown
	return nil
}

func main() {
	app := application.New(
		application.WithName("my-service"),
		application.WithVersion("1.0.0"),
		application.WithEnvironment("production"),
		application.WithGracefulTimeout(15*time.Second),
	)

	if err := app.Register(&HTTPServer{}); err != nil {
		slog.Error(err.Error())
	}

	ctx := context.Background()
	if err := app.Run(ctx); err != nil {
		slog.Error(err.Error())
	}
}
```

---

## 📄 Лицензия

Проект распространяется под лицензией [MIT](LICENSE).

---

## 🤝 Вклад в проект

PR и issue приветствуются! Следуйте стандартам форматирования и покрывайте новый код тестами.

---

> **Автор**: MSeytumerov  
> **Репозиторий**: `github.com/shuldan/app`  
> **Go**: `1.24.2`