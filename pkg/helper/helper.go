package helper

// Ptr is a helper function to create pointers from literals
func Ptr[T any](v T) *T {
	return &v
}
