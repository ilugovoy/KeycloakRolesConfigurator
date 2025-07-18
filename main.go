// main.go
package main

import (
	"context"
	"os"
	"os/signal"
)

// версия по умолчанию, заполняется при сборке
var version = "2.2.3"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	app := NewApp(version)

	go func() {
		<-ctx.Done()
		logInfo("Завершение по сигналу пользователя...")
		os.Exit(0)
	}()

	if err := app.Run(ctx); err != nil {
		logError("Ошибка при выполнении: ", err)
		os.Exit(1)
	}
}
