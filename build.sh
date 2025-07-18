#!/bin/bash

# Конфигурация
APP_NAME="KeycloakRolesConfigurator"
VERSION="${VERSION:-1.0}"
BUILD_DIR="build"

# Чтение учётных данных из auth.yaml
KEYCLOAK_USER=$(grep 'admin_user' auth.yaml | awk '{print $2}' | tr -d '"')
KEYCLOAK_PASS=$(grep 'admin_pass' auth.yaml | awk '{print $2}' | tr -d '"')

# Проверка наличия учётных данных
if [ -z "$KEYCLOAK_USER" ] || [ -z "$KEYCLOAK_PASS" ]; then
    echo "Ошибка: не удалось прочитать учётные данные из auth.yaml"
    exit 1
fi

# Очистка старых сборок
echo "Очистка старых файлов сборки..."
rm -rf ${BUILD_DIR} ${APP_NAME}_v* ${APP_NAME}_v*.exe

# Создание структуры директорий
mkdir -p ${BUILD_DIR}/{darwin,windows}

# Проверка наличия goimports
if ! command -v goimports &> /dev/null; then
    echo "Установка goimports..."
    go install golang.org/x/tools/cmd/goimports@latest
fi

echo "Форматирование кода..."
goimports -w *.go

echo "Подготовка зависимостей..."
go mod tidy

echo "Сборка версии: $VERSION"

# Сборка для macOS
echo "1/2 Сборка для macOS (darwin/arm64)..."
env GOOS=darwin GOARCH=arm64 go build \
  -ldflags "-X main.version=$VERSION -X main.keycloakUser=$KEYCLOAK_USER -X main.keycloakPass=$KEYCLOAK_PASS" \
  -o "${BUILD_DIR}/darwin/${APP_NAME}_v${VERSION}"

# Сборка для Windows
echo "2/2 Сборка для Windows (windows/amd64)..."
env GOOS=windows GOARCH=amd64 go build \
  -ldflags "-X main.version=$VERSION -X main.keycloakUser=$KEYCLOAK_USER -X main.keycloakPass=$KEYCLOAK_PASS" \
  -o "${BUILD_DIR}/windows/${APP_NAME}_v${VERSION}.exe"

# Проверка успешности сборки
if [ $? -eq 0 ]; then
    echo ""
    echo "Сборка успешно завершена!"
    echo "Результаты:"
    echo "  macOS:   ${BUILD_DIR}/darwin/${APP_NAME}_v${VERSION}"
    echo "  Windows: ${BUILD_DIR}/windows/${APP_NAME}_v${VERSION}.exe"
    echo ""
else
    echo ""
    echo "Ошибка сборки!"
    exit 1
fi