// role.go предоставляет функционал для управления ролями в Keycloak
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/schollz/progressbar/v3"
)

// Role представляет структуру роли в Keycloak
type Role struct {
	Name string `json:"name"`
}

// RoleResponse содержит ответ API при создании роли/группы
type RoleResponse struct {
	ID string `json:"id"`
}

// Assign используется для назначения ролей
type Assign struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// createRole создает новую роль в Keycloak
func (app *Operation) createRole(roleName string) {
	body := Role{Name: roleName}
	resp, err := app.client.R().SetBody(body).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"instance": app.realm,
			"clientId": app.clientId,
		}).Post("/admin/realms/{instance}/clients/{clientId}/roles")

	if err != nil || resp.StatusCode() != http.StatusCreated {
		bodyString := resp.String()
		if !strings.Contains(bodyString, "already exists") {
			app.AddError(fmt.Sprintf("Ошибка создания роли %s причина в %s, статус: %v",
				roleName, bodyString, resp.StatusCode()))
		}
	}
}

// createSubGroup создает подгруппу для роли
func (app *Operation) createSubGroup(group string) string {
	body := Role{Name: group}
	resp, err := app.client.R().SetBody(body).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"instance": app.realm,
			"group":    app.parentGroupId,
		}).Post("/admin/realms/{instance}/groups/{group}/children")

	if err != nil || resp.StatusCode() != http.StatusCreated {
		if resp.StatusCode() == http.StatusConflict {
			return app.getSubGroupByName(group)
		}
		app.AddError(fmt.Sprintf("Ошибка создания группы %s причина в %s, статус %v",
			group, resp.String(), resp.StatusCode()))
		return ""
	}

	var result RoleResponse
	json.Unmarshal(resp.Body(), &result)
	return result.ID
}

// getSubGroupByName ищет подгруппу по имени
func (app *Operation) getSubGroupByName(groupName string) string {
	resp, err := app.client.R().SetPathParams(map[string]string{
		"instance": app.realm,
		"group":    app.parentGroupId,
	}).Get("/admin/realms/{instance}/groups/{group}")

	if err != nil || resp.StatusCode() != http.StatusOK {
		app.AddError(fmt.Sprintf("Не удалось найти группу %s причина %d",
			groupName, resp.StatusCode()))
		return ""
	}

	var result Group
	json.Unmarshal(resp.Body(), &result)

	res, err := app.client.R().
		SetPathParam("realm", app.realm).
		SetPathParam("groupId", result.ID).
		SetQueryParam("first", "0").
		SetQueryParam("max", "250").
		Get("/admin/realms/{realm}/groups/{groupId}/children")

	if err != nil {
		app.AddError(fmt.Sprintf("Ошибка запроса подгрупп: %v", err))
		return ""
	}
	if res.StatusCode() != http.StatusOK {
		app.AddError(fmt.Sprintf("HTTP %d: не удалось получить подгруппы", res.StatusCode()))
		return ""
	}

	var subGroups []Group
	json.Unmarshal(res.Body(), &subGroups)

	for _, group := range subGroups {
		if group.Name == groupName {
			return group.ID
		}
	}
	return ""
}

// assignRole назначает роль группе
func (app *Operation) assignRole(role, id, groupId string) {
	assign := []Assign{{Name: role, ID: id}}
	res, err := app.client.R().SetBody(assign).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"instance": app.realm,
			"groupId":  groupId,
			"clientId": app.clientId,
		}).Post("/admin/realms/{instance}/groups/{groupId}/role-mappings/clients/{clientId}")

	if err != nil || res.StatusCode() != http.StatusNoContent {
		app.AddError(fmt.Sprintf("Ошибка при назначении роли на шаге 2 группе %s вызвана %d",
			groupId, res.StatusCode()))
	}
}

// addMember добавляет пользователя в группу
func (app *Operation) addMember(userId, groupId string) {
	resp, err := app.client.R().SetPathParams(map[string]string{
		"instance": app.realm,
		"userId":   userId,
		"groupId":  groupId,
	}).Put("/admin/realms/{instance}/users/{userId}/groups/{groupId}")

	if err != nil || resp.StatusCode() != http.StatusNoContent {
		app.AddError(fmt.Sprintf("Ошибка при добавлении участника %s в группу %s вызвана %d",
			userId, groupId, resp.StatusCode()))
	}
}

// processRole обрабатывает роль в зависимости от действия
func (app *Operation) processRole(bar *progressbar.ProgressBar) {
	roleId := app.findRole(app.roleName, false)
	subGroupId := app.getSubGroupByName(app.roleName)

	if app.action == "Create new role and add users to this role" {
		if roleId != "" || subGroupId != "" {
			log.Printf("Роль %s уже существует, смена действия на 'Associate users with role'", app.roleName)
			app.action = "Associate users with role"
		} else {
			app.createRole(app.roleName)
			roleId = app.findRole(app.roleName, true)
			subGroupId = app.createSubGroup(app.roleName)
			if roleId == "" || subGroupId == "" {
				return
			}
		}
	}

	switch app.action {
	case "Associate users with role":
		if roleId == "" || subGroupId == "" {
			app.AddError(fmt.Sprintf("Роль %s не существует. Пропуск LDAP:%s", app.roleName, app.ldapsString))
			return
		}
		app.assignRoleWithGroup(roleId, subGroupId, bar)
	case "Remove users from role":
		if roleId == "" || subGroupId == "" {
			app.AddError(fmt.Sprintf("Роль %s не существует. Пропуск LDAP:%s", app.roleName, app.ldapsString))
			return
		}
		app.removeUsersFromGroup(subGroupId, bar)
	}
}

// assignRoleWithGroup назначает роль и добавляет пользователей в группу
func (app *Operation) assignRoleWithGroup(roleId string, subGroupId string, bar *progressbar.ProgressBar) {
	app.assignRole(app.roleName, roleId, subGroupId)

	for _, ldap := range app.ldaps {
		userId := app.getUserIdByLdap(ldap)
		if userId == "" {
			continue
		}
		app.addMember(userId, subGroupId)
		_ = bar.Add(1)
	}
}

// removeUsersFromGroup удаляет пользователей из группы
func (app *Operation) removeUsersFromGroup(subGroupId string, bar *progressbar.ProgressBar) {
	for _, ldap := range app.ldaps {
		userId := app.getUserIdByLdap(ldap)
		if userId == "" {
			continue
		}
		app.removeMember(userId, subGroupId)
		_ = bar.Add(1)
	}
}

// removeMember удаляет пользователя из группы
func (app *Operation) removeMember(userId, groupId string) {
	resp, err := app.client.R().SetPathParams(map[string]string{
		"instance": app.realm,
		"userId":   userId,
		"groupId":  groupId,
	}).Delete("/admin/realms/{instance}/users/{userId}/groups/{groupId}")

	if err != nil || resp.StatusCode() != http.StatusNoContent {
		app.AddError(fmt.Sprintf("Ошибка удаления участника %s из группы %s вызвана %d",
			userId, groupId, resp.StatusCode()))
	}
}

// findRole ищет роль по имени с возможностью повторных попыток
func (app *Operation) findRole(roleName string, create bool) string {
	get, err := app.client.AddRetryCondition(
		func(r *resty.Response, err error) bool {
			return (r.StatusCode() == http.StatusNotFound) && create
		},
	).SetRetryCount(5).
		SetRetryMaxWaitTime(20 * time.Second).
		SetRetryWaitTime(4 * time.Second).
		R().SetPathParams(map[string]string{
		"instance": app.realm,
		"clientId": app.clientId,
		"role":     roleName,
	}).Get("/admin/realms/{instance}/clients/{clientId}/roles/{role}")

	if (err != nil || get.StatusCode() != http.StatusOK) && create {
		app.AddError(fmt.Sprintf("Не удается найти роль %s причина %s, вызвана %v",
			roleName, get.String(), get.StatusCode()))
		return ""
	}

	var result RoleResponse
	json.Unmarshal(get.Body(), &result)
	return result.ID
}
