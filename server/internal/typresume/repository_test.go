package typresume

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestResumeJSONDistinguishesMissingAndEmptyAssets(t *testing.T) {
	var missing Resume
	if err := json.Unmarshal([]byte(`{"id":"one"}`), &missing); err != nil {
		t.Fatal(err)
	}
	if missing.Assets != nil {
		t.Fatal("omitted assets should stay nil for old-client compatibility")
	}

	var empty Resume
	if err := json.Unmarshal([]byte(`{"id":"one","assets":{}}`), &empty); err != nil {
		t.Fatal(err)
	}
	if empty.Assets == nil {
		t.Fatal("explicit empty assets should remain non-nil")
	}
}

func TestAssetsFromContentSupportsOldClients(t *testing.T) {
	got := assetsFromContent([]byte(`{"basics":{"avatarBase64":"data:image/png;base64,AA=="}}`))
	if got["avatar.png"] != "data:image/png;base64,AA==" {
		t.Fatalf("unexpected assets: %#v", got)
	}

	if got := assetsFromContent([]byte(`{"basics":{}}`)); got != nil {
		t.Fatalf("expected nil assets, got %#v", got)
	}
}

func TestNormalizeInitializesAssets(t *testing.T) {
	in := Resume{}
	normalize(&in)

	if in.Assets == nil {
		t.Fatal("assets should be initialized")
	}
}

func TestScanOneDecodesAssets(t *testing.T) {
	now := time.Now()
	scan := func(dest ...any) error {
		values := []any{
			"resume-1",
			"测试简历",
			"typst",
			[]byte(`{}`),
			"main.typ",
			[]byte(`{"main.typ":"Hello"}`),
			[]byte(`{"avatar.jpg":"data:image/jpeg;base64,AA=="}`),
			now,
			now,
		}
		for i, value := range values {
			reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(value))
		}
		return nil
	}

	got, err := scanOne(scan)
	if err != nil {
		t.Fatalf("scanOne returned error: %v", err)
	}
	if got.Assets["avatar.jpg"] != "data:image/jpeg;base64,AA==" {
		t.Fatalf("unexpected assets: %#v", got.Assets)
	}
}
