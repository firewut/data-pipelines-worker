package helpers

import (
	"errors"
)

// Define an interface that includes only ordered types like int, float64, etc.
type ordered interface {
	int | float64 | string
}

type BlockConfig[T ordered] struct {
	Condition string
	Data      T
	Value     T
}

func EvaluateCondition[T ordered](data, value T, condition string) (bool, error) {
	switch condition {
	case "==":
		return data == value, nil
	case "!=":
		return data != value, nil
	case ">":
		return GreaterThan(data, value)
	case "<":
		return LessThan(data, value)
	case ">=":
		return GreaterThanOrEqual(data, value)
	case "<=":
		return LessThanOrEqual(data, value)
	default:
		return false, errors.New("unsupported condition: " + condition)
	}
}

// Generic comparison functions for ordered types (int, float64, string)
func GreaterThan[T ordered](a, b T) (bool, error) {
	switch any(a).(type) {
	case int, float64, string:
		return a > b, nil
	default:
		return false, errors.New("unsupported type for '>'")
	}
}

func LessThan[T ordered](a, b T) (bool, error) {
	switch any(a).(type) {
	case int, float64, string:
		return a < b, nil
	default:
		return false, errors.New("unsupported type for '<'")
	}
}

func GreaterThanOrEqual[T ordered](a, b T) (bool, error) {
	result, err := GreaterThan(a, b)
	if err != nil {
		return false, err
	}
	if result {
		return true, nil
	}
	return a == b, nil
}

func LessThanOrEqual[T ordered](a, b T) (bool, error) {
	result, err := LessThan(a, b)
	if err != nil {
		return false, err
	}
	if result {
		return true, nil
	}
	return a == b, nil
}
