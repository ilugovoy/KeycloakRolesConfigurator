// file_utils.go содержит вспомогательные функции для работы с файлами
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
)

var (
	logFile *os.File
	logger  *log.Logger
)

func initLogger() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	logPath := filepath.Join(filepath.Dir(exePath), "keycloak_configurator.log")
	logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	consoleWriter := &colorWriter{w: os.Stdout}
	multiWriter := io.MultiWriter(consoleWriter, logFile)

	logger = log.New(multiWriter, "", log.LstdFlags)
	logger.Println("=== New session started ===")
	return nil
}

type colorWriter struct {
	w io.Writer
}

func (cw *colorWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	switch {
	case strings.Contains(msg, "ERROR"):
		msg = colorRed + msg + colorReset
	case strings.Contains(msg, "WARN"):
		msg = colorYellow + msg + colorReset
	}
	return cw.w.Write([]byte(msg))
}

func logInfo(format string, v ...interface{}) {
	logger.Printf("INFO "+format, v...)
}

func logWarn(format string, v ...interface{}) {
	logger.Printf("WARN - "+format, v...)
}

func logError(format string, v ...interface{}) {
	logger.Printf("ERROR - "+format, v...)
}

// findExcelFiles ищет все Excel-файлы в указанной директории
func findExcelFiles(dir string) ([]string, error) {
	var excelFiles []string

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения директории: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			ext := strings.ToLower(filepath.Ext(file.Name()))
			if ext == ".xlsx" || ext == ".xls" {
				fullPath := filepath.Join(dir, file.Name())
				excelFiles = append(excelFiles, fullPath)
			}
		}
	}

	return excelFiles, nil
}
