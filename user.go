// user.go предоставляет функционал для работы с пользователями Keycloak
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	usersEndpoint     = "/admin/realms/{instance}/users"
	userSearchTimeout = 5 * time.Second
)

// UserResponse представляет структуру ответа API Keycloak при запросе пользователя
type UserResponse struct {
	Id string `json:"id"`
}

// getUserIdByLdap ищет пользователя в Keycloak по LDAP-логину
func (app *Operation) getUserIdByLdap(ldap string) string {
	if err := app.checkRateLimit(ldap); err != nil {
		return ""
	}

	resp, err := app.searchUser(ldap)
	if err != nil {
		return ""
	}

	return app.parseUserResponse(resp, ldap)
}

// checkRateLimit проверяет ограничение частоты запросов
func (app *Operation) checkRateLimit(ldap string) error {
	ctx, cancel := context.WithTimeout(context.Background(), userSearchTimeout)
	defer cancel()

	if err := userSearchLimit.Wait(ctx); err != nil {
		app.AddError(fmt.Sprintf("Превышен лимит запросов для LDAP: %s", ldap))
		return err
	}
	return nil
}

// searchUser выполняет поиск пользователя в Keycloak
func (app *Operation) searchUser(ldap string) (*resty.Response, error) {
	resp, err := app.client.R().
		SetPathParam("instance", app.realm).
		SetQueryParams(app.getUserSearchParams(ldap)).
		Get(usersEndpoint)

	if err != nil || resp.StatusCode() != http.StatusOK {
		app.AddError(fmt.Sprintf("Ошибка поиска LDAP: %s, статус: %d",
			ldap, resp.StatusCode()))
		return nil, err
	}
	return resp, nil
}

// getUserSearchParams возвращает параметры поиска пользователя
func (app *Operation) getUserSearchParams(ldap string) map[string]string {
	return map[string]string{
		"exact":    "true",
		"username": ldap,
	}
}

// parseUserResponse обрабатывает ответ от Keycloak
func (app *Operation) parseUserResponse(resp *resty.Response, ldap string) string {
	var users []UserResponse
	if err := json.Unmarshal(resp.Body(), &users); err != nil {
		app.AddError(fmt.Sprintf("Ошибка парсинга пользователя: %s", ldap))
		return ""
	}

	if len(users) == 0 {
		app.AddError(fmt.Sprintf("LDAP не найден: %s", ldap))
		return ""
	}

	return users[0].Id
}
