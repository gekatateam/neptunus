package slices

// Returns values from first that not in second
func Diff[T comparable](first, second []T) []T {
	var diff []T
	bucket := make(map[T]struct{}, len(second))
	for _, x := range second {
		bucket[x] = struct{}{}
	}
	for _, x := range first {
		if _, found := bucket[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

func Cross[T comparable](first, second []T) []T {
	var cross []T
	bucket := make(map[T]bool)
	for _, i := range first {
		for _, j := range second {
			if i == j && !bucket[i] {
				cross = append(cross, i)
				bucket[i] = true
			}
		}
	}
	return cross
}

func Unique[T comparable](s []T) []T {
	bucket := make(map[T]bool)
	list := []T{}
	for _, item := range s {
		if _, value := bucket[item]; !value {
			bucket[item] = true
			list = append(list, item)
		}
	}
	return list
}

func Contains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}
