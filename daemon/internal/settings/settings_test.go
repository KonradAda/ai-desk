package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsAreValid(t *testing.T) {
	if err := Defaults().Validate(); err != nil {
		t.Fatal(err)
	}
	if Defaults().Language != "en" {
		t.Fatal("English must be the default language")
	}
}

func TestValidateRejectsBadValues(t *testing.T) {
	for name, mut := range map[string]func(*Settings){
		"language":   func(s *Settings) { s.Language = "xx" },
		"mode":       func(s *Settings) { s.SessionMode = "sometimes" },
		"cards low":  func(s *Settings) { s.CardsPerPage = 0 },
		"cards high": func(s *Settings) { s.CardsPerPage = 7 },
		"hide":       func(s *Settings) { s.HideDoneAfterMin = -1 },
	} {
		s := Defaults()
		mut(&s)
		if s.Validate() == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	st := Load(path)
	want := Defaults()
	want.CardsPerPage = 4
	want.HideDoneAfterMin = 15
	want.SessionMode = "manual"
	if err := st.Set(want); err != nil {
		t.Fatal(err)
	}
	if got := Load(path).Get(); got != want {
		t.Fatalf("want %+v, got %+v", want, got)
	}
}

func TestLoadFallsBackOnBadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte(`{"cards_per_page": 99}`), 0o644)
	if got := Load(path).Get(); got != Defaults() {
		t.Fatalf("invalid file should give defaults, got %+v", got)
	}
}
