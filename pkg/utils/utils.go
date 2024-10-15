package utils

func ShallowPtrCopy[T any](s T) *T {
	c := new(T)
	*c = s
	return c
}
