# `app` — Фреймворк управления жизненным циклом Go-приложений

[![Go CI](https://github.com/shuldan/app/workflows/Go%20CI/badge.svg)](https://github.com/shuldan/app/actions)
[![codecov](https://codecov.io/gh/shuldan/app/branch/main/graph/badge.svg)](https://codecov.io/gh/shuldan/app)
[![Go Report Card](https://goreportcard.com/badge/github.com/shuldan/app)](https://goreportcard.com/report/github.com/shuldan/app)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

Минималистичный каркас для построения Go-приложений по принципам DDD с поддержкой модульной архитектуры, graceful shutdown, фоновых модулей, health-чеков и хуков жизненного цикла.

---

## Содержание

- [Основные возможности](#-основные-возможности)
- [Установка](#-установка)
- [Быстрый старт](#-быстрый-старт)
- [Архитектура](#-архитектура)
  - [Application](#application)
  - [Module](#module)
  - [BackgroundModule](#backgroundmodule)
  - [HealthChecker](#healthchecker)
  - [Hook](#hook)
  - [Logger](#logger)
- [Опции конфигурации](#-опции-конфигурации)
- [Контекст приложения](#-контекст-приложения)
- [Порядок выполнения](#-порядок-выполнения)
- [Обработка ошибок](#-обработка-ошибок)
- [Примеры использования](#-примеры-использования)
  - [Базовый модуль](#базовый-модуль)
  - [HTTP-сервер как фоновый модуль](#http-сервер-как-фоновый-модуль)
  - [Health-чеки](#health-чеки)
  - [Хуки жизненного цикла](#хуки-жизненного-цикла)
  - [Интеграция с slog](#интеграция-с-slog)
  - [Полный пример приложения](#полный-пример-приложения)
- [Инструменты разработки](#-инструменты-разработки)
- [Лицензия](#-лицензия)

---

## 🚀 Основные возможности

- **Модульная архитектура** — компоненты подключаются как реализации интерфейса `Module`
- **Graceful shutdown** — корректное завершение работы с настраиваемым таймаутом
- **Фоновые модули** — поддержка модулей с долгоживущими горутинами через `BackgroundModule`
- **Health-чеки** — агрегация состояния модулей через опциональный интерфейс `HealthChecker`
- **Хуки жизненного цикла** — внедрение кросс-модульной логики на этапах `BeforeStart`, `AfterStart`, `BeforeStop`, `AfterStop`
- **Обработка сигналов ОС** — автоматический перехват `SIGINT` и `SIGTERM`
- **Идемпотентность** — защита от повторного запуска и регистрации дублей
- **Валидация конфигурации** — ошибки конфигурации обнаруживаются при создании приложения
- **Абстракция логирования** — подключаемый логгер через интерфейс `Logger`
- **Контекст с метаданными** — имя, версия, окружение и время старта доступны через контекст

---

## 📦 Установка

```sh
go get github.com/shuldan/app
```

Требуется Go 1.24+.

---

## ⚡ Быстрый старт

```go
package main

import (
	"context"
	"fmt"

	"github.com/shuldan/app"
)

type greeter struct{}

func (g *greeter) Name() string                       { return "greeter" }
func (g *greeter) Init(_ context.Context) error        { return nil }
func (g *greeter) Start(_ context.Context) error       { fmt.Println("Hello!"); return nil }
func (g *greeter) Stop(_ context.Context) error        { fmt.Println("Bye!"); return nil }

func main() {
	a, err := app.New(
		app.WithName("my-service"),
		app.WithVersion("1.0.0"),
	)
	if err != nil {
		panic(err)
	}

	_ = a.Register(&greeter{})

	if err := a.Run(context.Background()); err != nil {
		panic(err)
	}
}
```

---

## 🧱 Архитектура

### Application

Центральный объект, управляющий жизненным циклом всех модулей.

```go
a, err := app.New(opts ...Option) (*Application, error)
```

Методы:

| Метод | Описание |
|-------|----------|
| `Register(module Module) error` | Регистрация модуля. Запрещена после вызова `Run` |
| `Run(ctx context.Context) error` | Запуск приложения. Блокирует до завершения |
| `Health(ctx context.Context) error` | Агрегированная проверка состояния всех `HealthChecker`-модулей |
| `Uptime() time.Duration` | Время работы приложения |

---

### Module

Базовый интерфейс компонента приложения. Каждый модуль обязан реализовать все четыре метода.

```go
type Module interface {
    Name() string
    Init(ctx context.Context) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}
```

| Метод | Фаза | Описание |
|-------|-------|----------|
| `Name()` | — | Уникальное имя модуля для логов, ошибок и дедупликации |
| `Init(ctx)` | Инициализация | Подготовка: чтение конфигурации, создание подключений, валидация |
| `Start(ctx)` | Запуск | Запуск рабочей логики модуля |
| `Stop(ctx)` | Остановка | Корректное завершение: закрытие подключений, сброс буферов |

Модули инициализируются и запускаются **в порядке регистрации**, останавливаются **в обратном порядке**.

---

### BackgroundModule

Расширение `Module` для компонентов с долгоживущими горутинами (HTTP-серверы, gRPC-серверы, Kafka-консьюмеры и т.д.).

```go
type BackgroundModule interface {
    Module
    Err() <-chan error
}
```

Если фоновый модуль отправляет ошибку в канал `Err()`, приложение автоматически инициирует graceful shutdown.

---

### HealthChecker

Опциональный интерфейс. Если модуль его реализует, он участвует в агрегированных health-чеках через `Application.Health()`.

```go
type HealthChecker interface {
    Health(ctx context.Context) error
}
```

Модуль может реализовать одновременно `Module` и `HealthChecker` — достаточно добавить метод `Health`.

---

### Hook

Структура для внедрения кросс-модульной логики на ключевых этапах жизненного цикла. Любое поле может быть `nil` — оно будет пропущено.

```go
type Hook struct {
    BeforeStart func(ctx context.Context) error
    AfterStart  func(ctx context.Context) error
    BeforeStop  func(ctx context.Context) error
    AfterStop   func(ctx context.Context) error
}
```

---

### Logger

Абстракция логирования. По умолчанию используется no-op логгер.

```go
type Logger interface {
    Info(msg string, args ...any)
    Error(msg string, args ...any)
}
```

Интерфейс совместим с `*slog.Logger` — его можно передать напрямую через `WithLogger`.

---

## ⚙️ Опции конфигурации

Все опции передаются при создании приложения. Возвращают ошибку при невалидных значениях.

```go
a, err := app.New(
    app.WithName("my-service"),            // обязательное, непустая строка
    app.WithVersion("1.0.0"),              // версия приложения
    app.WithEnvironment("production"),     // окружение
    app.WithGracefulTimeout(15*time.Second), // таймаут остановки (>= 0)
    app.WithLogger(slog.Default()),        // логгер
    app.WithHook(app.Hook{                 // хуки жизненного цикла
        BeforeStart: func(ctx context.Context) error { return nil },
    }),
)
```

| Опция | Значение по умолчанию | Валидация |
|-------|----------------------|-----------|
| `WithName(name)` | `""` | Не может быть пустой строкой |
| `WithVersion(version)` | `""` | — |
| `WithEnvironment(env)` | `""` | — |
| `WithGracefulTimeout(d)` | `10s` | Не может быть отрицательным. `0` — ожидание без ограничения |
| `WithLogger(logger)` | `noopLogger` | `nil` игнорируется |
| `WithHook(hook)` | — | Можно добавить несколько хуков |

---

## 🏷️ Контекст приложения

Метаданные приложения автоматически помещаются в контекст при старте. Для извлечения используются функции-аксессоры.

```go
func handler(ctx context.Context) {
    name    := app.NameFromContext(ctx)        // string
    version := app.VersionFromContext(ctx)     // string
    env     := app.EnvironmentFromContext(ctx) // string
    started := app.StartTimeFromContext(ctx)   // time.Time
}
```

Контекст передаётся во все методы модулей (`Init`, `Start`) и в хуки (`BeforeStart`, `AfterStart`). Для фазы остановки используется отдельный контекст с таймаутом.

---

## 📋 Порядок выполнения

### Запуск

```
1. Блокировка регистрации (registry.lock)
2. Обогащение контекста метаданными
3. Запуск обработчика сигналов ОС (горутина)
4. Init всех модулей (в порядке регистрации)
5. Хуки BeforeStart
6. Start всех модулей (в порядке регистрации)
7. Хуки AfterStart
8. Мониторинг BackgroundModule ошибок
9. Ожидание сигнала завершения
```

### Завершение

```
1. Хуки BeforeStop
2. Stop всех модулей (в обратном порядке регистрации)
3. Хуки AfterStop
```

### Поведение при ошибках запуска

Если `Start` модуля `N` вернул ошибку — все ранее успешно запущенные модули `[0..N-1]` будут остановлены в обратном порядке. Ошибки старта и остановки объединяются через `errors.Join`.

---

## ❌ Обработка ошибок

Все ошибки оборачиваются с указанием имени модуля для удобства отладки:

```
init module "database": connection refused
start module "http-server": bind: address already in use
stop module "cache": context deadline exceeded
background module "kafka-consumer": broker not available
```

### Предопределённые ошибки

| Ошибка | Описание |
|--------|----------|
| `ErrApplicationAlreadyRunning` | Повторный вызов `Run` |
| `ErrApplicationAlreadyStopped` | Приложение уже остановлено |
| `ErrGracefulShutdownTimedOut` | Модули не успели остановиться за `shutdownTimeout` |
| `ErrRegistrationClosed` | Попытка регистрации модуля после вызова `Run` |
| `ErrModuleAlreadyRegistered` | Модуль с таким именем уже зарегистрирован |
| `ErrModuleNameEmpty` | Имя модуля не может быть пустым |
| `ErrAppNameEmpty` | Имя приложения не может быть пустым |
| `ErrShutdownTimeoutNonPositive` | Таймаут остановки не может быть отрицательным |

Для проверки используйте `errors.Is`:

```go
if errors.Is(err, app.ErrGracefulShutdownTimedOut) {
    // принудительное завершение
}
```

---

## 📚 Примеры использования

### Базовый модуль

Минимальная реализация компонента — например, модуля кэша:

```go
type CacheModule struct {
    client *redis.Client
}

func (c *CacheModule) Name() string { return "cache" }

func (c *CacheModule) Init(ctx context.Context) error {
    c.client = redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    return c.client.Ping(ctx).Err()
}

func (c *CacheModule) Start(_ context.Context) error {
    return nil // кэш не требует запуска горутины
}

func (c *CacheModule) Stop(_ context.Context) error {
    return c.client.Close()
}
```

---

### HTTP-сервер как фоновый модуль

Модули с долгоживущими горутинами реализуют `BackgroundModule`. Если сервер аварийно завершится, приложение автоматически инициирует shutdown.

```go
type HTTPModule struct {
    server *http.Server
    errCh  chan error
}

func NewHTTPModule(addr string, handler http.Handler) *HTTPModule {
    return &HTTPModule{
        server: &http.Server{Addr: addr, Handler: handler},
        errCh:  make(chan error, 1),
    }
}

func (h *HTTPModule) Name() string { return "http-server" }

func (h *HTTPModule) Init(_ context.Context) error {
    return nil
}

func (h *HTTPModule) Start(_ context.Context) error {
    go func() {
        if err := h.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            h.errCh <- err
        }
        close(h.errCh)
    }()
    return nil
}

func (h *HTTPModule) Stop(ctx context.Context) error {
    return h.server.Shutdown(ctx)
}

func (h *HTTPModule) Err() <-chan error {
    return h.errCh
}
```

---

### Health-чеки

Модуль может реализовать `HealthChecker` для участия в агрегированных проверках состояния:

```go
type DatabaseModule struct {
    db *sql.DB
}

func (d *DatabaseModule) Name() string { return "database" }

func (d *DatabaseModule) Init(ctx context.Context) error {
    var err error
    d.db, err = sql.Open("postgres", "postgres://localhost/mydb")
    if err != nil {
        return err
    }
    return d.db.PingContext(ctx)
}

func (d *DatabaseModule) Start(_ context.Context) error { return nil }

func (d *DatabaseModule) Stop(_ context.Context) error { return d.db.Close() }

// HealthChecker
func (d *DatabaseModule) Health(ctx context.Context) error {
    return d.db.PingContext(ctx)
}
```

Проверка состояния приложения:

```go
// например, в обработчике /healthz
func healthHandler(a *app.Application) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if err := a.Health(r.Context()); err != nil {
            http.Error(w, err.Error(), http.StatusServiceUnavailable)
            return
        }
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
    }
}
```

---

### Хуки жизненного цикла

Хуки позволяют внедрить логику без создания отдельного модуля:

```go
a, _ := app.New(
    app.WithName("my-service"),
    app.WithHook(app.Hook{
        BeforeStart: func(ctx context.Context) error {
            slog.Info("preparing to start",
                "app", app.NameFromContext(ctx),
                "version", app.VersionFromContext(ctx),
            )
            return nil
        },
        AfterStart: func(_ context.Context) error {
            slog.Info("all modules started, ready to serve traffic")
            return nil
        },
        BeforeStop: func(_ context.Context) error {
            slog.Info("draining connections...")
            return nil
        },
        AfterStop: func(_ context.Context) error {
            slog.Info("flushing metrics...")
            return nil
        },
    }),
)
```

Хуки можно добавлять несколько раз — они выполняются в порядке регистрации.

---

### Интеграция с slog

Интерфейс `Logger` совместим с `*slog.Logger` из стандартной библиотеки:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

a, _ := app.New(
    app.WithName("my-service"),
    app.WithLogger(logger),
)
```

При желании можно адаптировать любой логгер под интерфейс:

```go
type Logger interface {
    Info(msg string, args ...any)
    Error(msg string, args ...any)
}
```

---

### Полный пример приложения

```go
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/shuldan/app"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	a, err := app.New(
		app.WithName("order-service"),
		app.WithVersion("2.1.0"),
		app.WithEnvironment("production"),
		app.WithGracefulTimeout(30*time.Second),
		app.WithLogger(logger),
		app.WithHook(app.Hook{
			AfterStart: func(_ context.Context) error {
				logger.Info("application is ready")
				return nil
			},
		}),
	)
	if err != nil {
		logger.Error("failed to create application", "error", err)
		os.Exit(1)
	}

	// Database
	db := &DatabaseModule{}
	if err := a.Register(db); err != nil {
		logger.Error("failed to register module", "error", err)
		os.Exit(1)
	}

	// Cache
	if err := a.Register(&CacheModule{}); err != nil {
		logger.Error("failed to register module", "error", err)
		os.Exit(1)
	}

	// HTTP Server
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := a.Health(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/uptime", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(a.Uptime().String()))
	})

	httpModule := NewHTTPModule(":8080", mux)
	if err := a.Register(httpModule); err != nil {
		logger.Error("failed to register module", "error", err)
		os.Exit(1)
	}

	// Run
	if err := a.Run(context.Background()); err != nil {
		if errors.Is(err, app.ErrGracefulShutdownTimedOut) {
			logger.Error("shutdown timed out, forcing exit")
			os.Exit(1)
		}
		logger.Error("application exited with error", "error", err)
		os.Exit(1)
	}
}
```

**Порядок событий при работе этого примера:**

```
1. Init:  database → cache → http-server
2. Hook:  BeforeStart (если есть)
3. Start: database → cache → http-server
4. Hook:  AfterStart → "application is ready"
5. Ожидание SIGINT/SIGTERM или ошибки BackgroundModule
6. Hook:  BeforeStop (если есть)
7. Stop:  http-server → cache → database
8. Hook:  AfterStop (если есть)
```

---

## 🛠️ Инструменты разработки

### Установка инструментов

```sh
make install-tools
```

Устанавливает:
- `golangci-lint` (v2.4.0)
- `goimports`
- `gosec`

### Основные команды

| Команда | Описание |
|---------|----------|
| `make all` | Форматирование + линтер + security-скан + тесты |
| `make ci` | CI-пайплайн: fmt + vet + lint + тесты с покрытием |
| `make fmt` | Форматирование кода и сортировка импортов |
| `make test` | Запуск тестов |
| `make test-coverage` | Тесты с отчётом о покрытии |

---

## 📄 Лицензия

Проект распространяется под лицензией [MIT](LICENSE).

---

## 🤝 Вклад в проект

PR и issue приветствуются. Требования:

1. Покрывайте новый код тестами
2. Запускайте `make all` перед отправкой PR
3. Следуйте существующему стилю кода

---

> **Автор**: MSeytumerov
> **Репозиторий**: `github.com/shuldan/app`
> **Go**: `1.24.2`
