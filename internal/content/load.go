package content

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/goccy/go-yaml"
)

// Load reads and validates the content tree rooted at dir.
//
// All problems found are reported together, joined with errors.Join, so an
// author fixing content sees every issue in one build instead of one per
// run. A non-nil error means nothing should be published.
func Load(dir string) (*Content, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	c := &Content{Root: root}

	if err := readYAML(filepath.Join(root, "site.yaml"), &c.Site); err != nil {
		return nil, err
	}
	if err := readYAML(filepath.Join(root, "profile.yaml"), &c.Profile); err != nil {
		return nil, err
	}
	if len(c.Site.Languages) == 0 {
		return nil, errors.New("site.yaml: languages must list at least one language")
	}

	projects, err := loadProjects(filepath.Join(root, "projects"), c.Site.Languages)
	if err != nil {
		return nil, err
	}
	c.Projects = projects

	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// loadProjects reads every directory under dir that contains a meta.yaml.
// Directories without one are ignored, which allows drafts to be kept as
// plain notes until they are ready.
func loadProjects(dir string, langs []Lang) ([]*Project, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var (
		projects []*Project
		errs     []error
	)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pdir := filepath.Join(dir, e.Name())
		meta := filepath.Join(pdir, "meta.yaml")
		if _, err := os.Stat(meta); errors.Is(err, fs.ErrNotExist) {
			continue
		}

		p := &Project{Slug: e.Name(), Dir: pdir, Body: map[Lang]string{}}
		if err := readYAML(meta, p); err != nil {
			errs = append(errs, err)
			continue
		}
		for _, l := range langs {
			b, err := os.ReadFile(filepath.Join(pdir, string(l)+".md"))
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				errs = append(errs, err)
				continue
			}
			p.Body[l] = string(b) // a missing file is reported by Validate
		}
		projects = append(projects, p)
	}

	sort.SliceStable(projects, func(i, j int) bool {
		if projects[i].Order != projects[j].Order {
			return projects[i].Order < projects[j].Order
		}
		return projects[i].Slug < projects[j].Slug
	})
	return projects, errors.Join(errs...)
}

// readYAML decodes path into v strictly: unknown keys are errors, so a typo
// such as "sumary:" fails the build instead of silently dropping content.
func readYAML(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.UnmarshalWithOptions(b, v, yaml.Strict()); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
