// Package content defines the site's content model and loads it from the
// content/ directory.
//
// The content tree is the single source of truth for everything a visitor
// reads. Code in this package is deliberately free of presentation concerns:
// it parses, normalises and validates, and hands plain Go values to the
// build package, which owns HTML.
package content

import (
	"fmt"
	"regexp"

	"github.com/goccy/go-yaml"
)

// Lang is a BCP 47 language tag as used in URLs and the html lang attribute
// (for example "tr" or "en").
type Lang string

// Text is a localised string.
//
// In YAML a Text may be written either as a plain scalar, which then applies
// to every language (useful for proper nouns, product names and numbers), or
// as a mapping from language tag to string:
//
//	period: "2026"                      # same in every language
//	title: { tr: "Deneyim", en: "Experience" }
//
// The scalar form is stored under the wildcard key anyLang so that the
// lookup in In stays a single map access plus one fallback.
type Text map[Lang]string

// anyLang is the key under which a language-neutral scalar is stored.
const anyLang Lang = "*"

// UnmarshalYAML implements yaml.BytesUnmarshaler so that both the scalar
// and the mapping form described on Text are accepted.
func (t *Text) UnmarshalYAML(b []byte) error {
	var scalar string
	if err := yaml.Unmarshal(b, &scalar); err == nil {
		*t = Text{anyLang: scalar}
		return nil
	}
	var m map[Lang]string
	if err := yaml.Unmarshal(b, &m); err != nil {
		return fmt.Errorf("expected a string or a {lang: string} mapping: %w", err)
	}
	// In YAML flow style a comma separates entries, so an unquoted
	// { tr: Bir, iki, en: … } silently becomes three keys. Rejecting
	// anything that is not a language tag turns that into a build error.
	// The split-off pieces become keys with empty values, which also
	// catches short fragments that happen to look like language tags.
	for k, v := range m {
		if !langTag.MatchString(string(k)) || v == "" {
			return fmt.Errorf("%q is not a language tag with a value; quote values that contain commas", k)
		}
	}
	*t = Text(m)
	return nil
}

// langTag matches the primary language subtags used as Text keys.
var langTag = regexp.MustCompile(`^[a-z]{2,3}$`)

// In returns the text for language l, falling back to the language-neutral
// value. It returns the empty string when neither exists; callers that need
// the value to be present should rely on Validate having run.
func (t Text) In(l Lang) string {
	if s, ok := t[l]; ok {
		return s
	}
	return t[anyLang]
}

// Has reports whether a non-empty value exists for l, directly or through
// the language-neutral fallback.
func (t Text) Has(l Lang) bool { return t.In(l) != "" }
