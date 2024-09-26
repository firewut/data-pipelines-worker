package helpers

import (
	"fmt"
	"strings"
)

func GetListAsQuotedString[T any](list []T) string {
	var quotedList []string
	for _, item := range list {
		quotedList = append(quotedList, fmt.Sprintf("\"%v\"", item))
	}
	return strings.Join(quotedList, ", ")
}
