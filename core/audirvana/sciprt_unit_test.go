package audirvana

import "testing"

func TestParseAudirvanaDuration(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    int64
		wantErr bool
	}{
		{name: "integer", raw: "284", want: 284},
		{name: "float", raw: "297.21600000000001", want: 297},
		{name: "float with comma", raw: "297,6", want: 298},
		{name: "empty", raw: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAudirvanaDuration(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseAudirvanaDuration(%q) expected error", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseAudirvanaDuration(%q) unexpected error: %v", tt.raw, err)
			}
			if got != tt.want {
				t.Fatalf("parseAudirvanaDuration(%q) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}
