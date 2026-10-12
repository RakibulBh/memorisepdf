package server

import (
	"strings"
	"testing"

	"github.com/RakibulBh/unmemorize/internal/store"
)

func TestNormaliseGenerateRequest(t *testing.T) {
	cases := []struct {
		name        string
		in          generateRequest
		wantKind    string
		wantVariant string
		wantOK      bool
	}{
		{
			name:        "quiz defaults to medium",
			in:          generateRequest{Kind: "quiz"},
			wantKind:    store.KindQuiz,
			wantVariant: "medium",
			wantOK:      true,
		},
		{
			name:        "quiz keeps an explicit difficulty",
			in:          generateRequest{Kind: "quiz", Variant: "hard"},
			wantKind:    store.KindQuiz,
			wantVariant: "hard",
			wantOK:      true,
		},
		{
			name:   "quiz rejects an unknown difficulty",
			in:     generateRequest{Kind: "quiz", Variant: "nightmare"},
			wantOK: false,
		},
		{
			// A stray variant on a flashcard request has to be dropped. Keeping
			// it would make a second ledger row for work already done, and the
			// same cards would be generated and paid for twice.
			name:        "flashcards discard any variant",
			in:          generateRequest{Kind: "flashcard", Variant: "hard"},
			wantKind:    store.KindFlashcard,
			wantVariant: "",
			wantOK:      true,
		},
		{
			name:        "teaching defaults to simple language",
			in:          generateRequest{Kind: "teaching"},
			wantKind:    store.KindTeaching,
			wantVariant: "simple-language",
			wantOK:      true,
		},
		{
			name:   "teaching rejects an unknown style",
			in:     generateRequest{Kind: "teaching", Variant: "interpretive-dance"},
			wantOK: false,
		},
		{
			name:   "an unknown kind is refused",
			in:     generateRequest{Kind: "podcast"},
			wantOK: false,
		},
		{
			name:   "an empty kind is refused",
			in:     generateRequest{},
			wantOK: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kind, variant, ok := normaliseGenerateRequest(c.in)

			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if !c.wantOK {
				return
			}
			if kind != c.wantKind {
				t.Errorf("kind = %q, want %q", kind, c.wantKind)
			}
			if variant != c.wantVariant {
				t.Errorf("variant = %q, want %q", variant, c.wantVariant)
			}
		})
	}
}

// The redirect parameter must not be usable to bounce somebody to another site
// after they sign in.
func TestSafeRedirect(t *testing.T) {
	s := &Server{}

	cases := map[string]string{
		"/dashboard":            "/dashboard",
		"/subjects/abc/modules": "/subjects/abc/modules",
		"":                      "",
		"//evil.example":        "",
		"https://evil.example":  "",
		"http://evil.example":   "",
		"dashboard":             "",
		"\\\\evil.example":      "",
	}

	for in, want := range cases {
		if got := s.safeRedirect(in); got != want {
			t.Errorf("safeRedirect(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCleanName(t *testing.T) {
	cases := []struct {
		in       string
		wantName string
		wantOK   bool
	}{
		{"Databases", "Databases", true},
		{"  Operating Systems  ", "Operating Systems", true},
		{"", "", false},
		{"   ", "", false},
		{strings.Repeat("a", maxNameLength), strings.Repeat("a", maxNameLength), true},
		{strings.Repeat("a", maxNameLength+1), "", false},
		{"bad\x00name", "", false},
		{"bad\tname", "", false},
	}

	for _, c := range cases {
		name, ok := cleanName(c.in)
		if ok != c.wantOK {
			t.Errorf("cleanName(%q) ok = %v, want %v", c.in, ok, c.wantOK)
			continue
		}
		if ok && name != c.wantName {
			t.Errorf("cleanName(%q) = %q, want %q", c.in, name, c.wantName)
		}
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		512:       "512 B",
		1 << 10:   "1 KB",
		100 << 20: "100 MB",
		500 << 20: "500 MB",
		2 << 30:   "2 GB",
	}

	for in, want := range cases {
		if got := humanBytes(in); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", in, got, want)
		}
	}
}
