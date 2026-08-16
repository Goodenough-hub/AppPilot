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
		{"ok with folder", Item{Type: "bookmark", Title: "x", Folder: "Infini-AI"}, ""},
		{"folder too long", Item{Type: "bookmark", Title: "x", Folder: string(make([]byte, 201))}, "folder too long"},
		{"ok with icon", Item{Type: "bookmark", Title: "x", Icon: "https://example.com/f.png"}, ""},
		{"icon too long", Item{Type: "bookmark", Title: "x", Icon: string(make([]byte, 1001))}, "icon too long"},
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

func TestValidateFolder(t *testing.T) {
	cases := []struct {
		name    string
		folder  Folder
		wantErr string
	}{
		{"ok", Folder{Type: "bookmark", Name: "Infini-AI"}, ""},
		{"ok name exactly 200 chars", Folder{Type: "skill", Name: string(make([]byte, 200))}, ""},
		{"empty name", Folder{Type: "bookmark", Name: ""}, "name required"},
		{"name too long", Folder{Type: "bookmark", Name: string(make([]byte, 201))}, "name too long"},
		{"invalid type", Folder{Type: "note", Name: "x"}, "invalid type"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.folder.Validate()
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
