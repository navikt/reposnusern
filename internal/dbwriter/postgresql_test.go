package dbwriter

import "testing"

func TestSafeLicense(t *testing.T) {
	tests := []struct {
		name string
		in   *struct{ SpdxID string }
		want string
	}{
		{"nil input", nil, ""},
		{"valid license", &struct{ SpdxID string }{SpdxID: "MIT"}, "MIT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := safeLicense(tt.in)
			if got != tt.want {
				t.Errorf("safeLicense() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSafeString(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want string
	}{
		{"nil input", nil, ""},
		{"valid string", "hello", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := safeString(tt.in)
			if got != tt.want {
				t.Errorf("safeString() = %q, want %q", got, tt.want)
			}
		})
	}
}
