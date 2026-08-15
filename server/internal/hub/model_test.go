package hub

import (
	"testing"
)

func TestValidateItem(t *testing.T) {
	cases := []struct {
		name    string
		item    Item
		wantErr string
	}{
		{"ok bookmark", Item{Type: "bookmark", Title: "GitHub", URL: strPtr("https://github.com")}, ""},
		{"ok prompt", Item{Type: "prompt", Title: "Code review", Content: strPtr("Please review...")}, ""},
		{"ok skill", Item{Type: "skill", Title: "whisper.cpp", URL: strPtr("https://github.com/ggerganov/whisper.cpp"), Content: strPtr("desc")}, ""},
		{"empty title", Item{Type: "bookmark", Title: ""}, "title required"},
		{"ok title exactly 500 chars", Item{Type: "bookmark", Title: string(make([]byte, 500))}, ""},
		{"title too long", Item{Type: "bookmark", Title: string(make([]byte, 501))}, "title too long"},
		{"invalid type", Item{Type: "note", Title: "x"}, "invalid type"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.item.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("want nil, got %v", err)
				}
				return
			}
			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("want %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func strPtr(s string) *string { return &s }
