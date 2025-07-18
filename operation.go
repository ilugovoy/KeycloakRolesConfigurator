// operation.go содержит основную структуру Operation и связанные с ней методы
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"golang.org/x/time/rate"
)

// Operation представляет одну операцию для обработки в Keycloak
type Operation struct {
	client        *resty.Client
	ClientIdName  string
	realm         string
	action        string
	roleName      string
	ldaps         []string
	ldapsString   string
	clientId      string
	parentGroupId string
	errors        map[int]string
	errorCounter  int
}

// Глобальные переменные
var (
	clientIdCache   = make(map[string]string)
	userSearchLimit = rate.NewLimiter(rate.Every(time.Second), 10) // rate limiter
)

// AddError добавляет ошибку в коллекцию ошибок операции
func (o *Operation) AddError(error string) {
	o.errorCounter++
	o.errors[o.errorCounter] = error

	// Логируем ошибку в файл
	logFile.WriteString(fmt.Sprintf("%s: %s\n", time.Now().Format("2006-01-02 15:04:05"), error))
}

// PrintErrors выводит ошибки в консоль
func (o *Operation) printErrors() {
	if len(o.errors) == 0 {
		return
	}

	for _, errMsg := range o.errors {
		// Добавляем "WARN -" к сообщениям об ошибках LDAP
		if strings.Contains(errMsg, "LDAP не найден") {
			logWarn("%s", strings.TrimPrefix(errMsg, "ERROR: "))
		} else {
			logError("%s", strings.TrimPrefix(errMsg, "ERROR: "))
		}
	}
}
