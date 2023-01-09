package utils

func ListContains(slice []string, obj string) bool {
	for i := 0; i < len(slice); i++ {
		if slice[i] == obj {
			return true
		}
	}
	return false
}

func ListDelete(slice []string, obj string) (result []string) {
	for i := 0; i < len(slice); i++ {
		if slice[i] == obj {
			continue
		}
		result = append(result, slice[i])
	}
	return
}
