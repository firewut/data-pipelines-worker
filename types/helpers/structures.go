package helpers

import (
	"reflect"
)

func MergeStructs(left, right, result interface{}) {
	// This function merges two structs into a third one
	// The result struct will contain the values from both input structs
	// If a field is present in both input structs, the value from the right struct will be used
	// If a field is present in only one of the input structs, the value from that struct will be used
	// If a field is not present in either input struct, the field will be set to the zero value of the field type

	// Get the type of the structs
	// leftType := reflect.TypeOf(left).Elem()
	// rightType := reflect.TypeOf(right).Elem()
	resultType := reflect.TypeOf(result).Elem()

	// Get the value of the structs
	leftValue := reflect.ValueOf(left).Elem()
	rightValue := reflect.ValueOf(right).Elem()
	resultValue := reflect.ValueOf(result).Elem()

	// Iterate over the fields of the result struct
	for i := 0; i < resultType.NumField(); i++ {
		field := resultType.Field(i)
		fieldName := field.Name

		// Get the value of the field in the left struct
		leftField := leftValue.FieldByName(fieldName)
		if leftField.IsValid() {
			// Set the value of the field in the result struct to the value of the field in the left struct
			resultValue.FieldByName(fieldName).Set(leftField)
		}

		// Get the value of the field in the right struct
		rightField := rightValue.FieldByName(fieldName)

		// TODO: This condition must be passed as parameter to the function
		if rightField.IsValid() && !rightField.IsZero() {
			// Set the value of the field in the result struct to the value of the field in the right struct
			resultValue.FieldByName(fieldName).Set(rightField)
		}
	}

	// Set the result struct to the value of the result struct
	reflect.ValueOf(result).Elem().Set(resultValue)

}
