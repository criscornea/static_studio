package ssg

import (
	"reflect"
	"testing"
)

func TestMergeConfig(t *testing.T) {
	tests := []struct {
		name     string
		dst, src map[string]any
		want     map[string]any
	}{
		{"nil dst", nil, map[string]any{"A": 1}, map[string]any{"a": 1}},
		{"scalar replaced", map[string]any{"a": 1}, map[string]any{"a": 2}, map[string]any{"a": 2}},
		{"case-insensitive key", map[string]any{"contentdir": "x"}, map[string]any{"ContentDir": "y"}, map[string]any{"contentdir": "y"}},
		{"maps merge deeply",
			map[string]any{"m": map[string]any{"a": 1, "b": 2}},
			map[string]any{"M": map[string]any{"B": 3, "c": 4}},
			map[string]any{"m": map[string]any{"a": 1, "b": 3, "c": 4}}},
		{"list replaced", map[string]any{"l": []any{"a", "b"}}, map[string]any{"l": []any{"c"}}, map[string]any{"l": []any{"c"}}},
		{"map replaces scalar", map[string]any{"a": 1}, map[string]any{"a": map[string]any{"X": 1}}, map[string]any{"a": map[string]any{"x": 1}}},
		{"scalar replaces map", map[string]any{"a": map[string]any{"x": 1}}, map[string]any{"a": 1}, map[string]any{"a": 1}},
		{"yaml any-keyed map", map[string]any{"a": map[string]any{"x": 1}}, map[string]any{"a": map[any]any{"y": 2}}, map[string]any{"a": map[string]any{"x": 1, "y": 2}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeConfig(tt.dst, tt.src); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
