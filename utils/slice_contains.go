package utils

func SliceContains(s []string, value string) bool {
	for _, item := range s {
		if item == value {
			return true
		}
	}
	return false
}
