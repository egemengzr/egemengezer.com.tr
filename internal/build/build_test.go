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
		`"@type":"WebSite"`,
	} {
		if !strings.Contains(string(home), want) {
			t.Errorf("home page lacks %q", want)
		}
	}
	// The CV must open in a new tab so the reader keeps the site open.
	if n := strings.Count(string(home), `.pdf" target="_blank" rel="noopener"`); n != 3 {
		t.Errorf("want 3 CV links opening in a new tab, found %d", n)
	}
	// Every image needs a description; project thumbnails use cover_alt.
	if strings.Contains(string(home), `alt=""`) {
		t.Error(`home page has an image with alt=""`)
	}
	if strings.Contains(string(home), "<script src") {
		t.Error("the site must not load JavaScript")
	}
	for _, f := range []string{"_redirects", "_headers", "sitemap.xml", "robots.txt", "llms.txt", "404.html"} {
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

// TestInlineSVGStripsMetadata checks that editor and provenance metadata is
// dropped from inlined diagrams while the drawing itself is kept.
func TestInlineSVGStripsMetadata(t *testing.T) {
	f := filepath.Join(t.TempDir(), "d.svg")
	src := `<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" xmlns:x="urn:x" xmlns:xlink="http://www.w3.org/1999/xlink" width="10" height="10" viewBox="0 0 10 10">` +
		`<rect fill="#1F4FD1" width="10" height="10"/><metadata><x:manifest>AAAA</x:manifest></metadata></svg>`
	if err := os.WriteFile(f, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := inlineSVG(f, "diagram")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "metadata") || strings.Contains(got, "AAAA") || strings.Contains(got, "urn:x") {
		t.Errorf("metadata not stripped: %s", got)
	}
	if !strings.Contains(got, `fill="var(--accent)"`) || !strings.Contains(got, "xmlns:xlink") {
		t.Errorf("drawing lost or palette not applied: %s", got)
	}
}
