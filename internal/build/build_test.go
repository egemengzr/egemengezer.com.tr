package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBuildRealContent builds the actual site end to end. It needs libvips,
// which CI installs; locally it is skipped when vips is missing.
func TestBuildRealContent(t *testing.T) {
	if _, err := exec.LookPath("vips"); err != nil {
		t.Skip("libvips not installed")
	}
	root := filepath.Join("..", "..")
	for _, f := range []string{"Egemen_Gezer_CV_TR.pdf", "Egemen_Gezer_CV_EN.pdf"} {
		if _, err := os.Stat(filepath.Join(root, "static", "cv", f)); err != nil {
			t.Skipf("static/cv/%s not present", f)
		}
	}
	out := t.TempDir()
	rep, err := Build(Options{
		ContentDir: filepath.Join(root, "content"),
		StaticDir:  filepath.Join(root, "static"),
		AssetsDir:  filepath.Join(root, "assets"),
		OutDir:     out,
		CacheDir:   filepath.Join(root, ".cache", "img"),
		Now:        time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Pages) == 0 {
		t.Fatal("no pages built")
	}

	home, err := os.ReadFile(filepath.Join(out, "tr", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<html lang="tr">`,
		`hreflang="en"`,
		`fetchpriority="high"`,
		`application/ld+json`,
	} {
		if !strings.Contains(string(home), want) {
			t.Errorf("home page lacks %q", want)
		}
	}
	// The CV must open in a new tab so the reader keeps the site open.
	if n := strings.Count(string(home), `.pdf" target="_blank" rel="noopener"`); n != 3 {
		t.Errorf("want 3 CV links opening in a new tab, found %d", n)
	}
	if strings.Contains(string(home), "<script src") {
		t.Error("the site must not load JavaScript")
	}
	for _, f := range []string{"_redirects", "_headers", "sitemap.xml", "robots.txt", "404.html"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}
}

func TestLoadCSSRewritesAssetURLs(t *testing.T) {
	css, err := loadCSS(map[string]string{
		"/assets/fonts/plex-mono-400.woff2": "/assets/fonts/plex-mono-400.abc.woff2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(css), "plex-mono-400.abc.woff2") || strings.Contains(string(css), "/*") {
		t.Errorf("unexpected CSS output: %.200s", css)
	}
}
