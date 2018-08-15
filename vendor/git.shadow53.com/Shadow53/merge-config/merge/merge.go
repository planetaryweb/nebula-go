package merge

// CopySlice creates a copy of a slice of interface{}, including copies of any
// maps or slices inside it.
func CopySlice(s []interface{}) []interface{} {
	var copy []interface{}
	for _, val := range s {
		switch v := val.(type) {
		case map[string]interface{}:
			copy = append(copy, CopyMap(v))
		case []interface{}:
			copy = append(copy, CopySlice(v))
		default:
			copy = append(copy, val)
		}
	}
	return copy
}

// CopyMap creates a copy of a map[string]interface{}, including copies of any
// maps or slices inside it.
func CopyMap(m map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{})
	for key, val := range m {
		switch v := val.(type) {
		case map[string]interface{}:
			copy[key] = CopyMap(v)
		case []interface{}:
			copy[key] = CopySlice(v)
		default:
			copy[key] = v
		}
	}
	return copy
}

// Merge creates a copy of the first argument and copies values from the second into it.
// If values in the second map are of reference type (map or slice), copies are made
// before assigning them to the copied map.
func Merge(l map[string]interface{}, r map[string]interface{}) map[string]interface{} {
	copy := CopyMap(l)
	for key, val := range r {
		switch v := val.(type) {
		case map[string]interface{}:
			if copy[key] != nil {
				if lmap, ok := copy[key].(map[string]interface{}); ok {
					copy[key] = Merge(lmap, v)
				} else {
					copy[key] = CopyMap(v)
				}
			} else {
				copy[key] = CopyMap(v)
			}
		case []interface{}:
			copy[key] = CopySlice(v)
		default:
			copy[key] = val
		}
	}
	return copy
}
