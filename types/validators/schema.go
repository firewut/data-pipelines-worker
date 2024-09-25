package validators

type FileFormatChecker struct{}

func (f FileFormatChecker) IsFormat(input interface{}) bool {
	// var str string

	// switch v := input.(type) {
	// case string:
	// 	str = v
	// case []byte:
	// 	str = string(v)
	// case *bytes.Buffer:
	// 	str = v.String()
	// default:
	// 	return false
	// }

	// // Move this pattern to a config file if needed
	// pattern := `^[a-zA-Z0-9_\-]+\.(txt|csv|mp3|mp4)$`
	// matched, err := regexp.MatchString(pattern, str)
	// if err != nil {
	// 	return false
	// }

	// return matched

	return true
}
