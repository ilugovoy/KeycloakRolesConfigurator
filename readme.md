# Keycloak Employees Roles Configurator 

[![Keycloak](https://img.shields.io/badge/Keycloak-2C7FBF?style=for-the-badge&logo=keycloak&logoColor=white)](https://www.keycloak.org/) [![Excel](https://img.shields.io/badge/Excel-217346?style=for-the-badge&logo=microsoftexcel&logoColor=white)](https://www.microsoft.com/excel) [![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://golang.org/) [![REST API](https://img.shields.io/badge/REST_API-005571?style=for-the-badge&logo=rest)](https://en.wikipedia.org/wiki/REST)  

<!-- TOC tocDepth:2..3 chapterDepth:2..6 -->
- [Keycloak Employees Roles Configurator](#keycloak-employees-roles-configurator)
	- [Как работает скрипт](#как-работает-скрипт)
		- [Особенности](#особенности)
	- [Быстрый старт](#быстрый-старт)
	- [Сборка](#сборка)
	- [Зависимости](#зависимости)
		- [Основные зависимости (direct)](#основные-зависимости-direct)
		- [Косвенные зависимости (indirect)](#косвенные-зависимости-indirect)
	- [Основные файлы](#основные-файлы)
	- [Вспомогательные файлы](#вспомогательные-файлы)
		- [go.mod](#gomod)
		- [go.sum](#gosum)
		- [build.sh](#buildsh)
		- [fix\_docs.sh](#fix_docssh)
	- [Обновление документации](#обновление-документации)
		- [Рекомендации](#рекомендации)
	- [TODO](#todo)
	- [Поддержка](#поддержка)
	- [Лицензия](#лицензия)
<!-- /TOC -->


## Как работает скрипт
Скрипт читает Excel-файлы с данными:
* Проверяет наличие листа `Request` с содержимым:  
  * `Environment` (`Prod/Dev/Test`)
  * `Instance` (`Employee/Partner/Customer`)
  * `Action` (`Create/Associate/Remove`)
  * `Role name`
  * `LDAPs` (через запятую)

Для каждой операции:
* Аутентифицируется в Keycloak
* Находит или создает клиента
* Находит или создает группу
* Выполняет выбранное действие:
    * Создает роль и добавляет пользователей
    * Связывает пользователей с существующей ролью
    * Удаляет пользователей из роли
    * Выводит прогресс и ошибки
* Пишет события в лог

### Особенности  
* Поддержка Excel (XLSX) вместо CSV
* Три типа действий с ролями:
    * `Create new role and add users to this role`
    * `Associate users with role`
    * `Remove users from role`
* Автоматическое определение URL Keycloak
* Прогресс-бар для операций
* Улучшенная обработка ошибок
* Поддержка версионирования

**Логирование**  
Программа создает файл `keycloak_configurator.log` в той же директории, где находится исполняемый файл.  

Формат логов:  
[дата] [уровень] сообщение
Уровни: `INFO`, `WARN`, `ERROR`


## Быстрый старт
1. Склонируйте репозиторий
2. Настройте `auth.yaml`, `excel_script.go` и соберите проект (см. раздел "Сборка")
3. Положите Excel-файлы в папку с бинарником и запустите программу


**Примечание**: Excel-файлы должны находиться в одной папке с исполяемым файлом скрипта, например:
```txt
├── KeycloakRolesConfigurator_v1.0.exe
├── roles_dev.xlsx
├── roles_prod.xlsx
└── history/
```
Нужно запустить скрипт и он обработает Excel-файлы.
**Примечание**: образец Exel-файла [](./Authorization_Template.xlsx)  


## Сборка
Используйте `build.sh` для сборки под разные платформы: `./build.sh`   
После выполнения вы увидите:  
```
$ ./build.sh
Очистка старых файлов сборки...
Форматирование кода...
Подготовка зависимостей...
Сборка версии: 1.0
1/2 Сборка для macOS (darwin/arm64)...
2/2 Сборка для Windows (windows/amd64)...

Сборка успешно завершена!
Результаты:
  macOS:   build/darwin/KeycloakRolesConfigurator_v1.0
  Windows: build/windows/KeycloakRolesConfigurator_v1.0.exe
``` 

**Примечание**: для аутентификации в Keycloak скрипт читает данные из файла `auth.yaml`. Для корректной работы внесите в файл `auth.yaml` свои учётные данные:
```yaml
keycloak:
  admin_user: "<ваш логин админской УЗ для доступа к Keycloak>"
  admin_pass: "<ваш пароль админской УЗ для доступа к Keycloak>"
```

**Примечание**: Для корректной работы внесите в файл `excel_script.go` эндпоинты своих инстансов Keycloak:
```go
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
```


## Зависимости
Описаны в `go.mod` 

* Косвенные зависимости (indirect) подключаются автоматически
* Все зависимости используют семантическое версионирование

### Основные зависимости (direct)

Пакет | Версия | Назначение
------|--------|-----------
github.com/go-resty/resty/v2 | v2.12.0 | HTTP-клиент для работы с API Keycloak
github.com/xuri/excelize/v2 | v2.8.1 | Чтение/запись Excel-файлов (XLSX)
github.com/schollz/progressbar/v3 | v3.14.2 | Интерактивный прогресс-бар для CLI
golang.org/x/time | v0.5.0 | Утилиты работы со временем (rate limiting)
gopkg.in/yaml.v3 | v3.0.1 | Парсинг YAML-конфигов


### Косвенные зависимости (indirect)
**Для Excelize**  

Пакет | Назначение
------|-----------
github.com/mohae/deepcopy | Глубокое копирование структур
github.com/richardlehane/mscfb | Чтение формата MS Compound File
github.com/richardlehane/msoleps | Парсинг OLE-потоков
github.com/xuri/efp | Парсинг формул Excel
github.com/xuri/nfp | Нормализация чисел в Excel

**Для Progressbar**  

Пакет | Назначение
------|-----------
github.com/mitchellh/colorstring | Поддержка цветного текста
github.com/rivo/uniseg | Работа с юникод-графемами

**Системные зависимости**  

Пакет | Назначение
------|-----------
golang.org/x/crypto | Криптографические функции
golang.org/x/net | Сетевые утилиты
golang.org/x/sys | Системные вызовы
golang.org/x/text | Утилиты работы с текстом


## Основные файлы  
Описаны в `DOCUMENTATION.md`  

* `main.go` - точка входа
* `app.go` - содержит основную логику приложения
* `operation.go` - основная структура
* `authentication.go` - логика аутентификации
* `client.go` - работа с Keycloak API
* `role.go` - управление ролями
* `user.go` - операции с пользователями
* `excel_script.go` - обработка Excel
* `file_utils.go` - логика логирования


## Вспомогательные файлы

### go.mod
Файл описания модуля Go

Зависимости:
* `resty/v2` - HTTP-клиент для работы с API Keycloak
* `progressbar/v3` - отображение прогресса выполнения
* `excelize/v2` - чтение/запись Excel-файлов

**Функции**:
- Фиксация версий зависимостей
- Управление модулями Go


### go.sum
Файл контрольных сумм зависимостей

Содержит:
* Точные версии всех зависимостей
* Контрольные суммы для проверки целостности
* Транзитивные (косвенные) зависимости

Ключевые зависимости:
* `golang.org/x/net` - сетевые utilities
* `golang.org/x/text` - обработка текста
* `golang.org/x/crypto` - криптографические функции
* `github.com/stretchr/testify` - тестирование (косвенная)

Функция:
* Гарантирует повторяемость сборок
* Защищает от подмены зависимостей
* Содержит полный граф зависимостей

**Важно**:
- Не редактировать вручную
- Автоматически обновляется Go


### build.sh
Скрипт для кросс-платформенной сборки

**Функции**:
- Кросс-платформенная сборка (Windows/macOS)
- Автоматическое версионирование через `-ldflags`
- Генерация имен файлов с версиями

Использование:  
```bash
chmod +x build.sh
# для исправления импортов
go install golang.org/x/tools/cmd/goimports@latest
goimports -w *.go
# указать версию в файлеи запустить
./build.sh
``` 


### fix_docs.sh  
Скрипт для исправления якорей в документации:  
* Стандартизирует якоря заголовков
* Оптимизирует оглавления
* Чистит Markdown-разметку (ссылки в оглавлении)  

Использование:  
```bash
chmod +x fix_docs.sh
./fix_docs.sh
```


## Обновление документации
Документация сделана с gomarkdoc.   
Установка: `go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest`    
Восстановить зависимости: `go mod tidy`  
После зменнеий в коде выполнить `gomarkdoc -u --output DOCUMENTATION.md ./...`  
Синхронизировать якоря: `fix_docs.sh`  


### Рекомендации
* Запускайте gomarkdoc и `fix_docs.sh` после любых изменений в коде  
* Версионируйте документацию вместе с кодом  
* Для проверки используйте Markdown-просмотрщик  

**Note:**   
Для работы скриптов требуется:  
* Bash (Linux/macOS) или Git Bash (Windows)
* Go 1.22+
* Утилиты sed/grep


## TODO  
* Разнести компоненты по отдельным пакетам (например, app, utils, config и т.д.).


## Поддержка
Если вы обнаружили проблему, создайте `issue`.  
Для значительных изменений — `fork` и `pull request`.  


## Лицензия
MIT License. Подробности см. в файле [LICENSE](LICENSE).