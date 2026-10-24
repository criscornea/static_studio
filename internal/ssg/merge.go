package ssg

import "strings"

// mergeConfig deep-merges src into dst and returns dst. Keys compare
// case-insensitively and are stored lower-cased, as Hugo does. Maps merge
// recursively; any other value in src replaces the one in dst.
func mergeConfig(dst, src map[string]any) map[string]any {
	if dst == nil {
		dst = make(map[string]any, len(src))
	}
	for k, v := range src {
		k = strings.ToLower(k)
		sm, srcIsMap := lowerMap(v)
		dm, dstIsMap := dst[k].(map[string]any)
		if srcIsMap && dstIsMap {
			dst[k] = mergeConfig(dm, sm)
			continue
		}
		if srcIsMap {
			v = mergeConfig(nil, sm)
		}
		dst[k] = v
	}
	return dst
}

// lowerMap reports whether v is a string-keyed map. YAML decoders may
// produce map[string]any or map[any]any.
func lowerMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, val := range m {
			if s, ok := k.(string); ok {
				out[s] = val
			}
		}
		return out, true
	}
	return nil, false
}
