// app.go содержит основную логику приложения
package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/schollz/progressbar/v3"
)

// App представляет основное приложение
type App struct {
	version       string
	consoleLogger *log.Logger
}

// NewApp создает новый экземпляр приложения
func NewApp(version string) *App {
	return &App{
		version:       version,
		consoleLogger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// Run запускает основную логику приложения
func (a *App) Run(ctx context.Context) error {
	if err := initLogger(); err != nil {
		logError("Ошибка инициализации логгера: ", err)
		return err
	}
	defer logFile.Close()

	exeDir, err := os.Executable()
	if err != nil {
		logError("Ошибка определения пути к исполняемому файлу: ", err)
		return err
	}
	exeDir = filepath.Dir(exeDir)

	files, err := findExcelFiles(exeDir)
	if err != nil {
		logError("Ошибка поиска Excel-файлов: ", err)
		return err
	}

	if len(files) == 0 {
		logWarn("Не найдено Excel-файлов для обработки")
		logInfo("Нажмите Enter для выхода...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		return nil
	}

	return a.processFiles(files)
}

// processFiles обрабатывает найденные Excel-файлы
func (a *App) processFiles(files []string) error {
	logInfo("Запуск Keycloak Configurator версии %s", a.version)
	logInfo("Найдено %d Excel-файлов для обработки", len(files))
	logInfo("Список файлов:")
	for i, file := range files {
		logInfo("%2d. %s", i+1, filepath.Base(file))
	}

	hasErrors := false
	for _, file := range files {
		filename := filepath.Base(file)
		logInfo("Начинаем обработку файла: %s", filename)

		if err := a.processFile(file); err != nil {
			logError("Ошибка обработки файла %s: %v", filename, err)
			hasErrors = true
		}

		logInfo("Завершена обработка файла: %s", filename)
	}

	if hasErrors {
		logWarn("ВНИМАНИЕ: Были ошибки при обработке некоторых файлов!")
		logInfo("Проверьте файл keycloak_configurator.log для подробностей")
	}

	logInfo("Обработка всех файлов завершена")
	logInfo("Нажмите Enter для выхода...")
	bufio.NewReader(os.Stdin).ReadString('\n')
	return nil
}

// processFile обрабатывает один файл
func (a *App) processFile(file string) error {
	operations, err := readExcelFile(file)
	if err != nil {
		return err
	}

	if len(operations) == 0 {
		logWarn("Файл ", filepath.Base(file), " не содержит операций для обработки")
		return nil
	}

	for i, operation := range operations {
		if err := a.processOperation(&operation, i, len(operations)); err != nil {
			return err
		}
	}
	return nil
}

// processOperation обрабатывает одну операцию
func (a *App) processOperation(operation *Operation, index, total int) error {
	logInfo("Обработка операции %d/%d: %s - %s",
		index+1, total, operation.action, operation.roleName)

	bar := progressbar.Default(int64(len(operation.ldaps)))

	if err := operation.Authenticate(); err != nil {
		logError("Ошибка аутентификации для операции %s: %v", operation.roleName, err)
		operation.printErrors()
		return nil
	}

	if err := operation.FindClientIdByName(); err != nil {
		logError("Ошибка поиска клиента для операции ", operation.roleName, ": ", err)
		operation.printErrors()
		return nil
	}

	if err := operation.FindOrCreateGroupByName(); err != nil {
		logError("Ошибка работы с группами для операции ", operation.roleName, ": ", err)
		operation.printErrors()
		return nil
	}

	operation.processRole(bar)

	if len(operation.errors) > 0 {
		operation.printErrors()
	}
	return nil
}
