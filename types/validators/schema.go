package validators

import (
	"regexp"
)

type FileFormatChecker struct{}

func (f FileFormatChecker) IsFormat(input interface{}) bool {
	str, ok := input.(string)
	if !ok {
		return false
	}

	// TODO: Move this pattern to a config file
	pattern := `^[a-zA-Z0-9_\-]+\.(txt|csv|mp3|mp4)$`
	matched, err := regexp.MatchString(pattern, str)
	if err != nil {
		return false
	}

	return matched
}
