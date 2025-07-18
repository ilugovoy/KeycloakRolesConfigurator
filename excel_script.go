// excel_script.go предоставляет функционал для обработки Excel-файла
//   - Читает Excel-файл и преобразует его в массив Operation
//   - Разбивает строку LDAP-групп на массив
//   - Генерирует URL для Keycloak на основе входных параметров из Excel-файла
//   - Работает с инстансами: "Employee", "Partner", "Customer B2C/B2B"
//   - Работает с окружениями: "Production", "Preproduction", "Stage"
package main

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/xuri/excelize/v2"
)

// Константы для валидации входных данных
const (
	validTypes        = "Employee|Partner|Customer"
	validEnvironments = "Prod|Dev|Test"
	validActions      = "Create new role and add users to this role|Associate users with role|Remove users from role"
	excelSheetName    = "Request"
	minColumnsCount   = 6
)

// ExcelConfig содержит конфигурацию для работы с Excel
type ExcelConfig struct {
	FilePath   string
	SheetName  string
	HeaderRows int
	MinColumns int
}

// readExcelFile читает и парсит Excel-файл, преобразуя его в массив Operation
func readExcelFile(filePath string) ([]Operation, error) {
	config := ExcelConfig{
		FilePath:   filePath,
		SheetName:  excelSheetName,
		HeaderRows: 1,
		MinColumns: minColumnsCount,
	}

	rows, err := readExcelRows(config)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	return processExcelRows(rows), nil
}

// readExcelRows читает данные из Excel файла
func readExcelRows(config ExcelConfig) ([][]string, error) {
	f, err := excelize.OpenFile(config.FilePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer closeExcelFile(f)

	// Получаем список всех листов
	sheets := f.GetSheetList()
	sheetExists := false
	for _, sheet := range sheets {
		if sheet == config.SheetName {
			sheetExists = true
			break
		}
	}

	if !sheetExists {
		return nil, fmt.Errorf("лист '%s' не найден. Доступные листы: %v",
			config.SheetName, sheets)
	}

	rows, err := f.GetRows(config.SheetName)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения листа %s: %w", config.SheetName, err)
	}

	if len(rows) <= config.HeaderRows {
		return nil, fmt.Errorf("файл не содержит данных для обработки")
	}

	return rows[config.HeaderRows:], nil
}

// closeExcelFile безопасно закрывает файл Excel
func closeExcelFile(f *excelize.File) {
	if err := f.Close(); err != nil {
		log.Printf("Ошибка при закрытии файла Excel: %v", err)
	}
}

// processExcelRows обрабатывает строки Excel и преобразует их в операции
func processExcelRows(rows [][]string) []Operation {
	var operations []Operation

	for i, row := range rows {
		operation, err := createOperationFromRow(row, i+2)
		if err != nil {
			log.Printf("Строка %d: %v - пропущена", i+2, err)
			continue
		}
		operations = append(operations, operation)
	}

	return operations
}

// createOperationFromRow создает Operation из строки Excel
func createOperationFromRow(row []string, rowNum int) (Operation, error) {
	if err := validateExcelRow(row, rowNum); err != nil {
		return Operation{}, err
	}

	baseURL, realm := getURLAndRealm(row[0], row[1])
	ldaps := parseLDAPs(row[5])

	return Operation{
		client:       createHTTPClient(baseURL),
		ClientIdName: row[3],
		realm:        realm,
		action:       row[2],
		roleName:     row[4],
		ldaps:        ldaps,
		ldapsString:  row[5],
		errors:       make(map[int]string),
		errorCounter: 0,
	}, nil
}

// validateExcelRow проверяет корректность строки из Excel-файла
func validateExcelRow(row []string, rowNum int) error {
	if len(row) < minColumnsCount {
		return fmt.Errorf("WARN - строка %d содержит только %d колонок", rowNum, len(row))
	}

	// Проверка пустых обязательных полей
	requiredFields := []struct {
		index int
		name  string
	}{
		{0, "Keycloak type"},
		{1, "Keycloak environment"},
		{2, "Action"},
		{3, "Client ID"},
		{4, "Role name"},
		{5, "User logins"},
	}

	for _, f := range requiredFields {
		if strings.TrimSpace(row[f.index]) == "" {
			errMsg := fmt.Sprintf("строка %d: %s не может быть пустым", rowNum, f.name)
			logWarn(errMsg)
			return errors.New(errMsg) // Самый чистый вариант
		}
	}

	// Валидация значений
	validators := []struct {
		index  int
		regex  string
		errMsg string
	}{
		{0, validTypes, "неверный тип Keycloak: %s. Допустимые: %s"},
		{1, validEnvironments, "неверное окружение: %s. Допустимые: %s"},
		{2, validActions, "неверное действие: %s. Допустимые: %s"},
	}

	for _, v := range validators {
		if !regexp.MustCompile(v.regex).MatchString(row[v.index]) {
			return fmt.Errorf("WARN - "+v.errMsg, row[v.index], v.regex)
		}
	}

	return nil
}

// createHTTPClient создает и настраивает HTTP-клиент
func createHTTPClient(baseURL string) *resty.Client {
	return resty.New().
		SetBaseURL(baseURL).
		SetHeader("Content-Type", "Application/x-www-form-urlencoded")
}

// parseLDAPs разбивает строку LDAP-групп на массив
func parseLDAPs(ldaps string) []string {
	if ldaps == "" {
		return nil
	}

	splitStr := strings.Split(ldaps, ",")
	result := make([]string, 0, len(splitStr))

	for _, str := range splitStr {
		if trimmed := strings.TrimSpace(str); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// getURLAndRealm генерирует URL и realm для Keycloak
func getURLAndRealm(instance, environment string) (string, string) {
	urlBuilder := strings.Builder{}
	urlBuilder.WriteString("https://")

	realm := getRealmByInstance(instance)
	domain := getDomainByInstanceAndEnv(instance, environment)

	urlBuilder.WriteString(domain)
	return urlBuilder.String(), realm
}

// getRealmByInstance возвращает realm по типу инстанса
func getRealmByInstance(instance string) string {
	switch instance {
	case "Employee":
		return "employee"
	case "Partner":
		return "partner"
	case "Customer":
		return "customer"
	default:
		return ""
	}
}

// getDomainByInstanceAndEnv возвращает домен по типу инстанса и окружению
func getDomainByInstanceAndEnv(instance, environment string) string {
	switch instance {
	case "Employee":
		switch environment {
		case "Prod":
			return "employee.your_domain.ru"
		case "Dev":
			return "employee-dev.your_domain.ru"
		case "Test":
			return "employee-test.your_domain.ru"
		}
	case "Partner":
		switch environment {
		case "Prod":
			return "partners.your_domain.ru"
		case "Dev":
			return "partners-dev.your_domain.ru"
		case "Test":
			return "partners-test.your_domain.ru"
		}
	case "Customer":
		switch environment {
		case "Prod":
			return "customer.your_domain.ru"
		case "Dev":
			return "customer-dev.your_domain.ru"
		case "Test":
			return "customer-test.your_domain.ru"
		}
	}
	return ""
}
