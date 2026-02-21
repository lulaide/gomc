package gen

import "testing"

func TestParseWorldType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		version int32
		want    WorldType
	}{
		{name: "default", input: "default", version: 1, want: WorldTypeDefault},
		{name: "default v0 maps to 1.1", input: "default", version: 0, want: WorldTypeDefault11},
		{name: "large biomes", input: "largeBiomes", version: 0, want: WorldTypeLargeBiomes},
		{name: "default 1.1 by name", input: "default_1_1", version: 0, want: WorldTypeDefault11},
		{name: "fallback unknown", input: "unknown", version: 999, want: WorldTypeDefault},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseWorldType(tc.input, tc.version)
			if got != tc.want {
				t.Fatalf("ParseWorldType(%q,%d)=%v want=%v", tc.input, tc.version, got, tc.want)
			}
		})
	}
}
