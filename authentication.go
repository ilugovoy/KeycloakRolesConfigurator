// authentication.go предоставляет инструмент для настройки ролей в Keycloak
//   - Аутентификация в Keycloak (OAuth2 Password Grant)
//   - Управление иерархией групп и ролей
//   - Импорт/экспорт настроек из Excel
package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
)

// Данные будут встроены при сборке
var (
	keycloakUser string
	keycloakPass string
)

// Session представляет структуру ответа от сервера аутентификации Keycloak
//   - Содержит все поля, возвращаемые при успешной аутентификации
type Session struct {
	AccessToken      string `json:"access_token"`       // Основной токен доступа
	ExpiresIn        int    `json:"expires_in"`         // Время жизни токена в секундах
	RefreshExpiresIn int    `json:"refresh_expires_in"` // Время жизни refresh-токена
	RefreshToken     string `json:"refresh_token"`      // Токен для обновления access token
	TokenType        string `json:"token_type"`         // Тип токена ("Bearer")
	NotBeforePolicy  int    `json:"not-before-policy"`  // Время, до которого токен недействителен
	SessionState     string `json:"session_state"`      // Идентификатор сессии
	Scope            string `json:"scope"`              // Области действия токена
}

const (
	tokenEndpoint = "/realms/{instance}/protocol/openid-connect/token"
	adminClientID = "admin-cli"
	grantType     = "password"
)

// Authenticate выполняет аутентификацию в Keycloak с использованием учётных данных,
// заданных при сборке бинарника через -ldflags
func (app *Operation) Authenticate() error {
	if err := app.validateAuthParams(); err != nil {
		return err
	}

	res, err := app.sendAuthRequest()
	if err != nil {
		return app.handleAuthError(err, res)
	}

	session, err := app.parseAuthResponse(res)
	if err != nil {
		return err
	}

	app.configureClient(session)
	return nil
}

// validateAuthParams проверяет обязательные параметры для аутентификации
func (app *Operation) validateAuthParams() error {
	if app.realm == "" {
		return fmt.Errorf("realm не может быть пустым")
	}
	if app.client == nil {
		return fmt.Errorf("HTTP-клиент не инициализирован")
	}
	if keycloakUser == "" || keycloakPass == "" {
		return fmt.Errorf("учётные данные Keycloak не установлены (должны задаваться при сборке)")
	}
	return nil
}

// sendAuthRequest отправляет запрос на аутентификацию
func (app *Operation) sendAuthRequest() (*resty.Response, error) {
	res, err := app.client.R().
		SetPathParam("instance", app.realm).
		SetFormData(app.buildAuthFormData()).
		Post(tokenEndpoint)

	defer app.resetTokenOnError(err, res)

	return res, err
}

// buildAuthFormData создаёт данные для формы аутентификации
func (app *Operation) buildAuthFormData() map[string]string {
	return map[string]string{
		"client_id":  adminClientID,
		"grant_type": grantType,
		"username":   keycloakUser,
		"password":   keycloakPass,
	}
}

// resetTokenOnError сбрасывает токен при ошибке аутентификации
func (app *Operation) resetTokenOnError(err error, res *resty.Response) {
	if err != nil || res.StatusCode() != http.StatusOK {
		app.client.SetAuthToken("")
	}
}

// handleAuthError обрабатывает ошибки аутентификации
func (app *Operation) handleAuthError(err error, res *resty.Response) error {
	app.AddError(fmt.Sprintf("Ошибка аутентификации: %d. Пропускаем LDAP: %s",
		res.StatusCode(), app.ldapsString))
	return err
}

// parseAuthResponse парсит ответ от сервера аутентификации
func (app *Operation) parseAuthResponse(res *resty.Response) (*Session, error) {
	var session Session
	if err := json.Unmarshal(res.Body(), &session); err != nil {
		app.AddError(fmt.Sprintf("Ошибка парсинга ответа: %v", err))
		return nil, fmt.Errorf("ошибка парсинга: %w", err)
	}
	return &session, nil
}

// configureClient настраивает клиент для последующих запросов
func (app *Operation) configureClient(session *Session) {
	app.client.
		SetHeader("Content-Type", "Application/json").
		SetAuthToken(session.AccessToken)
}
