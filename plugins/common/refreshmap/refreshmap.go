package refreshmap

import "maps"

const (
	MapRefreshSize      = 1024
	MapRefreshThreshold = 0.60
)

func RefreshIfNeeded[K comparable, V any](m map[K]V, prevLen int) map[K]V {
	if isItTimeToRefresh(len(m), prevLen) {
		newmap := make(map[K]V, len(m))
		maps.Copy(newmap, m)
		return newmap
	}
	return m
}

func isItTimeToRefresh(currLen, prevLen int) bool {
	utilization := float64(currLen) / float64(prevLen)
	return prevLen > MapRefreshSize && utilization < MapRefreshThreshold
}
