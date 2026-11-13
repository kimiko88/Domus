package tuya_local

import "testing"

func TestValidDeviceID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{name: "valid alphanumeric", id: "bf1234567890abcd", want: true},
		{name: "valid separators", id: "device_01-test", want: true},
		{name: "empty", id: "", want: false},
		{name: "whitespace", id: "device 01", want: false},
		{name: "control character", id: "device\n01", want: false},
		{name: "too long", id: string(make([]byte, 129)), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validDeviceID(tt.id); got != tt.want {
				t.Fatalf("validDeviceID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}
