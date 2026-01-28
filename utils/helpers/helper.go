package helpers


// derefString is a helper function to safely dereference *string
func DerefString(s *string) string {
	if s != nil {
		return *s
	}
	return "<nil>"
}