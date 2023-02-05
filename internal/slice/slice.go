package slice

func Map[T, R any](list []T, f func(t T) R) []R {
	if f == nil {
		return make([]R, 0)
	}

	output := make([]R, 0, len(list))

	for idx := range list {
		output = append(output, f(list[idx]))
	}

	return output
}

func Filter[T comparable](arr []T, filterFn func(v T) bool) []T {
	output := make([]T, 0, len(arr))
	for _, v := range arr {
		if filterFn == nil || filterFn(v) {
			output = append(output, v)
		}
	}

	return output
}

func Find[T any](list []T, f func(t T) bool) (T, bool) {
	var found T
	for idx := range list {
		if ok := f(list[idx]); ok {
			return list[idx], true
		}
	}

	return found, false
}
