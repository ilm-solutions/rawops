package cmd

import (
	"reflect"
	"testing"
)

func TestParseShieldPorts(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:  "valid ports",
			input: "22,25565,19132",
			want:  []string{"22", "25565", "19132"},
		},
		{
			name:  "whitespace",
			input: " 22, 80,443 ",
			want:  []string{"22", "80", "443"},
		},
		{
			name:  "duplicates",
			input: "22,80,22,443,80",
			want:  []string{"22", "80", "443"},
		},
		{
			name:  "normalize leading zeros",
			input: "022,080",
			want:  []string{"22", "80"},
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "empty entry",
			input:   "22,,80",
			wantErr: true,
		},
		{
			name:    "zero port",
			input:   "0",
			wantErr: true,
		},
		{
			name:    "out of range",
			input:   "65536",
			wantErr: true,
		},
		{
			name:    "negative port",
			input:   "-22",
			wantErr: true,
		},
		{
			name:    "non numeric",
			input:   "abc",
			wantErr: true,
		},
		{
			name:    "trailing separator",
			input:   "22,",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseShieldPorts(tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("parseShieldPorts() error = %v, wantErr %v",
					err, tt.wantErr)
			}

			if err != nil {
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseShieldPorts() = %v, want %v",
					got, tt.want)
			}
		})
	}
}
