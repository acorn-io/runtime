package typed

func MapSlice[T any, K any](values []T, mapper func(T) K) []K {
	result := make([]K, 0, len(values))
	for _, value := range values {
		result = append(result, mapper(value))
	}
	return result
}
