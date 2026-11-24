package content

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// maxSourceImageBytes caps source images in the content tree. The site never
// serves anything wider than 1440px, so larger files only grow the git
// history. Export at most 1600px on the long side (JPEG quality ~88).
const maxSourceImageBytes = 1 << 20

// slugPattern restricts project directory names to what is safe and
// readable in a URL without escaping.
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Validate checks the loaded content for completeness and consistency.
//
// The rules encode what a reader would notice as broken: a missing
// translation, a dangling image reference, an empty heading. Everything is
// checked for every configured language, so adding a language to site.yaml
// immediately lists every string that still needs translating.
func (c *Content) Validate() error {
	var v validator
	s, p := &c.Site, &c.Profile

	if !strings.HasPrefix(s.BaseURL, "https://") || strings.HasSuffix(s.BaseURL, "/") {
		v.addf("site.yaml: base_url must start with https:// and have no trailing slash")
	}
	if s.HTMLBudgetBytes <= 0 {
		v.addf("site.yaml: html_budget_bytes must be positive")
	}
	for _, l := range s.Languages {
		v.text("site.yaml projects_path", s.ProjectsPath, l)
		for key, t := range s.UI {
			v.text("site.yaml ui."+key, t, l)
		}
	}

	if p.Name == "" || p.Email == "" {
		v.addf("profile.yaml: name and email are required")
	}
	v.file("profile.yaml photo", filepath.Join(c.Root, p.Photo), p.Photo != "")
	for _, l := range s.Languages {
		v.text("profile.yaml cv", p.CV, l)
		v.text("profile.yaml status", p.Status, l)
		v.text("profile.yaml title", p.Title, l)
		v.text("profile.yaml role", p.Role, l)
		v.text("profile.yaml summary", p.Summary, l)
		v.text("profile.yaml description", p.Description, l)
		for i, f := range p.Facts {
			v.text(fmt.Sprintf("profile.yaml facts[%d].label", i), f.Label, l)
			v.text(fmt.Sprintf("profile.yaml facts[%d].value", i), f.Value, l)
		}
		for i, f := range p.Skills {
			v.text(fmt.Sprintf("profile.yaml skills[%d].label", i), f.Label, l)
			v.text(fmt.Sprintf("profile.yaml skills[%d].value", i), f.Value, l)
		}
		for i, e := range p.Experience {
			v.text(fmt.Sprintf("profile.yaml experience[%d].period", i), e.Period, l)
			v.text(fmt.Sprintf("profile.yaml experience[%d].role", i), e.Role, l)
			v.text(fmt.Sprintf("profile.yaml experience[%d].text", i), e.Text, l)
		}
	}

	v.imageSizes(c.Root)

	for _, pr := range c.Projects {
		where := "projects/" + pr.Slug
		if !slugPattern.MatchString(pr.Slug) {
			v.addf("%s: directory name must be lower-case words joined by hyphens", where)
		}
		if len(pr.Tags) == 0 {
			v.addf("%s/meta.yaml: tags must not be empty", where)
		}
		if pr.Cover != "" {
			v.file(where+"/meta.yaml cover", filepath.Join(pr.Dir, pr.Cover), true)
		}
		if len(pr.Bars) > 0 && pr.BarsMax <= 0 {
			v.addf("%s/meta.yaml: bars_max must be set when bars are present", where)
		}
		for _, l := range s.Languages {
			v.text(where+"/meta.yaml context", pr.Context, l)
			v.text(where+"/meta.yaml period", pr.Period, l)
			v.text(where+"/meta.yaml title", pr.Title, l)
			v.text(where+"/meta.yaml summary", pr.Summary, l)
			if pr.Cover != "" {
				v.text(where+"/meta.yaml cover_alt", pr.CoverAlt, l)
			}
			if len(pr.Result) > 0 {
				v.text(where+"/meta.yaml result", pr.Result, l)
			}
			for i, b := range pr.Bars {
				v.text(fmt.Sprintf("%s/meta.yaml bars[%d].label", where, i), b.Label, l)
			}
			for i, k := range pr.Links {
				v.text(fmt.Sprintf("%s/meta.yaml links[%d].label", where, i), k.Label, l)
			}
			if strings.TrimSpace(pr.Body[l]) == "" {
				v.addf("%s/%s.md: missing or empty", where, l)
			}
		}
	}
	return v.err()
}

// validator accumulates problems so that Validate can report all of them.
type validator struct{ errs []error }

func (v *validator) addf(format string, args ...any) {
	v.errs = append(v.errs, fmt.Errorf(format, args...))
}

// text records a problem when t has no value for l.
func (v *validator) text(where string, t Text, l Lang) {
	if !t.Has(l) {
		v.addf("%s: missing %q text", where, l)
	}
}

// file records a problem when the field is unset or path does not exist.
// set reports whether the YAML field had a value at all.
func (v *validator) file(where, path string, set bool) {
	if !set {
		v.addf("%s: required", where)
		return
	}
	if _, err := os.Stat(path); err != nil {
		v.addf("%s: %v", where, err)
	}
}

// imageSizes records every raster image under root larger than
// maxSourceImageBytes.
func (v *validator) imageSizes(root string) {
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		switch strings.ToLower(filepath.Ext(p)) {
		case ".jpg", ".jpeg", ".png", ".webp":
		default:
			return nil
		}
		if info, err := d.Info(); err == nil && info.Size() > maxSourceImageBytes {
			rel, _ := filepath.Rel(root, p)
			v.addf("%s: %d KB; resize to at most 1600px on the long side (limit %d KB)",
				rel, info.Size()/1024, maxSourceImageBytes/1024)
		}
		return nil
	})
}

func (v *validator) err() error { return errors.Join(v.errs...) }
