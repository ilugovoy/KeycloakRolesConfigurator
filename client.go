// client.go предоставляет функционал для работы с Keycloak API
//   - Поиск клиентов по имени
//   - Управление иерархией групп
//   - Создание подгрупп для ролей
//
// client.go предоставляет функционал для работы с Keycloak API
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
)

// Client представляет клиента в Keycloak
type Client struct {
	ID       string `json:"id"`       // Внутренний UUID клиента
	ClientID string `json:"clientId"` // Публичный идентификатор (например "admin-cli")
	Name     string `json:"name"`     // Отображаемое имя
}

// Group представляет иерархическую структуру групп Keycloak
type Group struct {
	ID        string  `json:"id"`        // Уникальный ID группы
	Name      string  `json:"name"`      // Название группы
	SubGroups []Group `json:"subGroups"` // Рекурсивная структура подгрупп
}

// Константы для endpoint'ов и параметров
const (
	clientsEndpoint       = "/admin/realms/{instance}/clients"
	groupsEndpoint        = "/admin/realms/{realm}/groups"
	groupChildrenEndpoint = "/admin/realms/{realm}/groups/{groupId}/children"
	defaultMaxResults     = 100
	rolesGroupName        = "Roles"
)

// FindClientIdByName ищет клиента в Keycloak по имени и сохраняет его ID в Operation
func (app *Operation) FindClientIdByName() error {
	if cachedID, exists := clientIdCache[app.ClientIdName]; exists {
		app.clientId = cachedID
		return nil
	}

	clients, err := app.fetchClients()
	if err != nil {
		return err
	}

	if err := app.validateClientSearchResults(clients); err != nil {
		return err
	}

	app.cacheAndSetClientID(clients[0].ID)
	return nil
}

// fetchClients выполняет запрос к API Keycloak для поиска клиентов
func (app *Operation) fetchClients() ([]Client, error) {
	res, err := app.client.R().
		SetPathParam("instance", app.realm).
		SetQueryParams(app.buildClientSearchParams()).
		Get(clientsEndpoint)

	if err != nil {
		app.logClientSearchError(res)
		return nil, err
	}

	return app.parseClientsResponse(res)
}

// buildClientSearchParams создает параметры для поиска клиента
func (app *Operation) buildClientSearchParams() map[string]string {
	return map[string]string{
		"clientId": app.ClientIdName,
		"first":    "0",
		"max":      fmt.Sprintf("%d", defaultMaxResults),
		"search":   "true",
	}
}

// parseClientsResponse парсит ответ от API Keycloak
func (app *Operation) parseClientsResponse(res *resty.Response) ([]Client, error) {
	var clients []Client
	if err := json.Unmarshal(res.Body(), &clients); err != nil {
		app.AddError(fmt.Sprintf("Ошибка парсинга клиентов: %s. Пропускаем LDAP: %s",
			err, app.ldapsString))
		return nil, err
	}
	return clients, nil
}

// validateClientSearchResults проверяет результаты поиска клиента
func (app *Operation) validateClientSearchResults(clients []Client) error {
	switch len(clients) {
	case 0:
		app.AddError(fmt.Sprintf("Клиент %s не найден. Пропускаем LDAP:%s",
			app.ClientIdName, app.ldapsString))
		return fmt.Errorf("client not found")
	case 1:
		return nil
	default:
		app.AddError(fmt.Sprintf("Найдено несколько клиентов с именем: %s. Пропускаем LDAP:%s",
			app.ClientIdName, app.ldapsString))
		return fmt.Errorf("multiple clients found")
	}
}

// cacheAndSetClientID сохраняет ID клиента в кэш и структуру Operation
func (app *Operation) cacheAndSetClientID(clientID string) {
	clientIdCache[app.ClientIdName] = clientID
	app.clientId = clientID
}

// logClientSearchError логирует ошибку поиска клиента
func (app *Operation) logClientSearchError(res *resty.Response) {
	app.AddError(fmt.Sprintf("Ошибка поиска клиента: %d. Пропускаем LDAP: %s",
		res.StatusCode(), app.ldapsString))
}

// FindOrCreateGroupByName ищет или создает группу для ролей
func (app *Operation) FindOrCreateGroupByName() error {
	rolesGroup, err := app.findRolesGroup()
	if err != nil {
		return err
	}

	if subgroupID := app.findClientSubgroup(rolesGroup); subgroupID != "" {
		app.parentGroupId = subgroupID
		return nil
	}

	return app.createClientSubgroup(rolesGroup.ID)
}

// findRolesGroup ищет родительскую группу "Roles"
func (app *Operation) findRolesGroup() (*Group, error) {
	res, err := app.client.R().
		SetPathParam("realm", app.realm).
		SetQueryParams(app.buildRolesGroupSearchParams()).
		Get(groupsEndpoint)

	if err != nil {
		app.logGroupSearchError(res)
		return nil, err
	}

	return app.parseRolesGroupResponse(res)
}

// buildRolesGroupSearchParams создает параметры для поиска группы "Roles"
func (app *Operation) buildRolesGroupSearchParams() map[string]string {
	return map[string]string{
		"exact":  "true",
		"search": rolesGroupName,
		"global": "true",
		"max":    "21",
	}
}

// parseRolesGroupResponse парсит ответ с информацией о группе "Roles"
func (app *Operation) parseRolesGroupResponse(res *resty.Response) (*Group, error) {
	var groups []Group
	if err := json.Unmarshal(res.Body(), &groups); err != nil || len(groups) == 0 {
		app.logGroupSearchError(res)
		return nil, fmt.Errorf("failed to parse roles group")
	}
	return &groups[0], nil
}

// findClientSubgroup ищет подгруппу клиента в группе "Roles"
func (app *Operation) findClientSubgroup(rolesGroup *Group) string {
	subgroups, err := app.getGroupSubgroups(rolesGroup.ID)
	if err != nil {
		return ""
	}

	for _, subgroup := range subgroups {
		if subgroup.Name == app.ClientIdName {
			return subgroup.ID
		}
	}
	return ""
}

// getGroupSubgroups получает подгруппы указанной группы
func (app *Operation) getGroupSubgroups(groupID string) ([]Group, error) {
	res, err := app.client.R().
		SetPathParam("realm", app.realm).
		SetPathParam("groupId", groupID).
		SetQueryParams(map[string]string{
			"first": "0",
			"max":   "250",
		}).
		Get(groupChildrenEndpoint)

	if err != nil {
		return nil, err
	}

	var subgroups []Group
	err = json.Unmarshal(res.Body(), &subgroups)
	return subgroups, err
}

// createClientSubgroup создает новую подгруппу для клиента
func (app *Operation) createClientSubgroup(parentGroupID string) error {
	res, err := app.client.R().
		SetBody(Role{Name: app.ClientIdName}).
		SetHeader("Content-Type", "application/json;charset=UTF-8").
		SetPathParams(map[string]string{
			"instance": app.realm,
			"group":    parentGroupID,
		}).
		Post(groupChildrenEndpoint)

	if err != nil || res.StatusCode() != http.StatusCreated {
		app.logGroupCreationError(res)
		return errors.New("group creation failed")
	}

	var group Group
	if err := json.Unmarshal(res.Body(), &group); err != nil {
		return err
	}

	app.parentGroupId = group.ID
	return nil
}

// logGroupSearchError логирует ошибку поиска группы
func (app *Operation) logGroupSearchError(res *resty.Response) {
	app.AddError(fmt.Sprintf("Не удается получить группу ролей из keycloak: %d. Пропускаем LDAP:%s",
		res.StatusCode(), app.ldapsString))
}

// logGroupCreationError логирует ошибку создания группы
func (app *Operation) logGroupCreationError(res *resty.Response) {
	app.AddError(fmt.Sprintf("Не удается создать группу в keycloak по причине: %s, код ошибки: %v. Пропускаем LDAP:%s",
		res.String(), res.StatusCode(), app.ldapsString))
}
