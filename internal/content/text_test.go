package content

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestTextScalarAppliesToAllLanguages(t *testing.T) {
	var v struct{ T Text }
	if err := yaml.Unmarshal([]byte(`t: Pulse Systech`), &v); err != nil {
		t.Fatal(err)
	}
	for _, l := range []Lang{"tr", "en"} {
		if got := v.T.In(l); got != "Pulse Systech" {
			t.Errorf("In(%q) = %q", l, got)
		}
	}
}

func TestTextMapping(t *testing.T) {
	var v struct{ T Text }
	if err := yaml.Unmarshal([]byte(`t: { tr: Deneyim, en: Experience }`), &v); err != nil {
		t.Fatal(err)
	}
	if v.T.In("tr") != "Deneyim" || v.T.In("en") != "Experience" {
		t.Errorf("got %v", v.T)
	}
	if v.T.Has("de") {
		t.Error("Has(de) = true for a language that was not given")
	}
}

// An unquoted comma in YAML flow style splits the value into extra keys.
// This must fail loudly rather than publish a truncated title.
func TestTextRejectsUnquotedComma(t *testing.T) {
	var v struct{ T Text }
	err := yaml.Unmarshal([]byte(`t: { tr: Bir, iki, en: One }`), &v)
	if err == nil || !strings.Contains(err.Error(), "quote values") {
		t.Fatalf("want a quoting error, got %v", err)
	}
}
