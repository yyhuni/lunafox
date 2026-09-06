package application

// orderedLastByKey keeps the final input item for each already-validated
// persistence key. Returning winners in source order makes the operation
// deterministic without letting this shared mechanic define resource identity.
func orderedLastByKey[T any](items []T, key func(T) string) ([]T, int) {
	last := make(map[string]int, len(items))
	for index := range items {
		last[key(items[index])] = index
	}
	winners := make([]T, 0, len(last))
	duplicates := 0
	for index := range items {
		if last[key(items[index])] != index {
			duplicates++
			continue
		}
		winners = append(winners, items[index])
	}
	return winners, duplicates
}
