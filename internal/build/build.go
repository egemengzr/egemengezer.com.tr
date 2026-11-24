// Package build renders the content tree into a static site directory that
// Cloudflare Workers serves as static assets.
//
// The output is plain files: one index.html per page, hashed and
// immutable-cacheable assets under /assets, plus the _headers and
// _redirects control files that Workers static assets understand. Nothing in
// the output runs on the server or in the browser.
package build

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"egemengezer.com.tr/site/internal/content"
	"egemengezer.com.tr/site/internal/imaging"
)

//go:embed web/site.css web/templates/*.gohtml
var webFS embed.FS

// Options configures a build. Paths are relative to the working directory.
type Options struct {
	ContentDir string // content/: YAML, Markdown and source images
	StaticDir  string // static/: copied verbatim (favicon, CV, _headers)
	AssetsDir  string // assets/: copied with content-hashed names (fonts)
	OutDir     string // output directory; deleted and recreated
	CacheDir   string // image encode cache, kept between builds
	Now        time.Time
}

// Report summarises a successful build for the console.
type Report struct {
	Pages []PageStat
}

// PageStat is the size of one generated HTML page.
type PageStat struct {
	Path        string
	Bytes, Gzip int
}

// Build renders the whole site. It is all-or-nothing: on error the output
// directory may be incomplete and must not be deployed (CI never deploys a
// failed build).
func Build(opt Options) (*Report, error) {
	c, err := content.Load(opt.ContentDir)
	if err != nil {
		return nil, fmt.Errorf("content:\n%w", err)
	}

	if err := os.RemoveAll(opt.OutDir); err != nil {
		return nil, err
	}
	if err := copyTree(opt.StaticDir, opt.OutDir); err != nil {
		return nil, fmt.Errorf("static: %w", err)
	}
	manifest, err := copyHashed(opt.AssetsDir, opt.OutDir, "/assets")
	if err != nil {
		return nil, fmt.Errorf("assets: %w", err)
	}
	for _, l := range c.Site.Languages {
		cv := c.Profile.CV.In(l)
		if _, err := os.Stat(filepath.Join(opt.OutDir, filepath.FromSlash(cv))); err != nil {
			return nil, fmt.Errorf("profile.yaml cv (%s): %s must exist under %s", l, cv, opt.StaticDir)
		}
	}

	imgs, err := imaging.NewProcessor(opt.OutDir, "/assets/img", opt.CacheDir)
	if err != nil {
		return nil, err
	}

	css, err := loadCSS(manifest)
	if err != nil {
		return nil, err
	}
	tpl, err := parseTemplates()
	if err != nil {
		return nil, err
	}

	b := &builder{c: c, opt: opt, imgs: imgs, css: css, tpl: tpl, manifest: manifest}
	if err := b.run(); err != nil {
		return nil, err
	}
	return &Report{Pages: b.pages}, nil
}

// builder carries the state of one build.
type builder struct {
	c        *content.Content
	opt      Options
	imgs     *imaging.Processor
	css      template.CSS
	tpl      map[string]*template.Template
	manifest map[string]string

	photo   map[content.Lang]template.HTML // alt text differs per language
	ogImage string

	pages []PageStat
	urls  []string // for the sitemap
}

func (b *builder) run() error {
	c := b.c
	photoSrc := filepath.Join(c.Root, c.Profile.Photo)

	portrait, err := b.imgs.Process(photoSrc, "profile", imaging.Portrait)
	if err != nil {
		return err
	}
	b.photo = map[content.Lang]template.HTML{}
	for _, l := range c.Site.Languages {
		// The photo is the largest element on first paint, so it is
		// fetched eagerly and with high priority.
		b.photo[l] = picture(portrait, c.Profile.Name, true)
	}
	og, err := b.imgs.Process(photoSrc, "profile", imaging.OpenGraph)
	if err != nil {
		return err
	}
	b.ogImage = c.Site.BaseURL + og.Fallback().Largest().URL

	for _, l := range c.Site.Languages {
		projects, err := b.projectViews(l)
		if err != nil {
			return err
		}
		if err := b.renderHome(l, projects); err != nil {
			return err
		}
		for _, p := range projects {
			if err := b.renderProject(l, p); err != nil {
				return err
			}
		}
	}
	if err := b.renderNotFound(); err != nil {
		return err
	}
	return b.writeExtras()
}

// ---- URLs -------------------------------------------------------------

func (b *builder) homeURL(l content.Lang) string { return "/" + string(l) + "/" }

func (b *builder) projectURL(l content.Lang, slug string) string {
	return "/" + string(l) + "/" + b.c.Site.ProjectsPath.In(l) + "/" + slug + "/"
}

// ---- View models ------------------------------------------------------
//
// Templates receive these flattened, single-language structs rather than
// content types, so they never deal with Text lookups or image processing.

type langLink struct {
	Lang    content.Lang
	Label   string
	URL     string
	Current bool
}

type alternate struct {
	Lang content.Lang
	URL  string
}

type pageData struct {
	Lang        content.Lang
	Title       string
	Description string
	Canonical   string
	Alternates  []alternate
	XDefault    string
	OGType      string
	OGLocale    string
	OGImage     string
	JSONLD      template.JS
	CSS         template.CSS
	FontURL     string
	HomeURL     string
	Langs       []langLink
	UI          map[string]string
	Rev         string
	Year        int
	Profile     *profileView
	Projects    []*projectView
	Project     *projectView
}

type kv struct{ Label, Value string }

type experienceView struct{ Org, Period, Role, Text string }

type profileView struct {
	Name, Email, CV                    string
	GitHub, GitHubURL, LinkedInURL     string
	Status, Role, Summary, Description string
	Photo                              template.HTML
	Facts, Skills                      []kv
	Experience                         []experienceView
}

type barView struct {
	Label string
	Pct   string
}

type linkView struct{ Label, URL string }

type projectView struct {
	Slug, URL                               string
	Context, Period, Title, Summary, Result string
	Tags                                    []string
	Thumb, Cover, Body                      template.HTML
	Bars                                    []barView
	Links                                   []linkView
	Prev, Next                              *projectView
}

func (b *builder) profileView(l content.Lang) *profileView {
	p := &b.c.Profile
	v := &profileView{
		Name: p.Name, Email: p.Email, CV: p.CV.In(l),
		GitHub:      p.GitHub,
		GitHubURL:   "https://github.com/" + p.GitHub,
		LinkedInURL: "https://www.linkedin.com/in/" + p.LinkedIn + "/",
		Status:      p.Status.In(l), Role: p.Role.In(l),
		Summary: p.Summary.In(l), Description: p.Description.In(l),
		Photo: b.photo[l],
	}
	for _, f := range p.Facts {
		v.Facts = append(v.Facts, kv{f.Label.In(l), f.Value.In(l)})
	}
	for _, f := range p.Skills {
		v.Skills = append(v.Skills, kv{f.Label.In(l), f.Value.In(l)})
	}
	for _, e := range p.Experience {
		v.Experience = append(v.Experience, experienceView{e.Org, e.Period.In(l), e.Role.In(l), e.Text.In(l)})
	}
	return v
}

// projectViews builds the views for every project in language l, including
// processed images and rendered Markdown, and links neighbours for the
// previous/next navigation.
func (b *builder) projectViews(l content.Lang) ([]*projectView, error) {
	var out []*projectView
	for _, p := range b.c.Projects {
		v := &projectView{
			Slug: p.Slug, URL: b.projectURL(l, p.Slug),
			Context: p.Context.In(l), Period: p.Period.In(l), Title: p.Title.In(l),
			Summary: p.Summary.In(l), Result: p.Result.In(l), Tags: p.Tags,
		}
		if p.Cover != "" {
			src := filepath.Join(p.Dir, p.Cover)
			thumb, err := b.imgs.Process(src, p.Slug+"-cover", imaging.Thumb)
			if err != nil {
				return nil, err
			}
			// Thumbnails are decorative next to the visible title, hence
			// the empty alt; they load lazily because most are off-screen.
			v.Thumb = picture(thumb, "", false)
			cover, err := b.imgs.Process(src, p.Slug+"-cover", imaging.Content)
			if err != nil {
				return nil, err
			}
			v.Cover = picture(cover, v.Title, true)
		}
		for _, bar := range p.Bars {
			v.Bars = append(v.Bars, barView{bar.Label.In(l), fmt.Sprintf("%.1f", clampPct(bar.Value/p.BarsMax*100))})
		}
		for _, k := range p.Links {
			v.Links = append(v.Links, linkView{k.Label.In(l), k.URL})
		}
		body, err := renderMarkdown(p.Body[l], b.resolver(p))
		if err != nil {
			return nil, fmt.Errorf("projects/%s/%s.md: %w", p.Slug, l, err)
		}
		v.Body = template.HTML(body)
		out = append(out, v)
	}
	for i, v := range out {
		if i > 0 {
			v.Prev = out[i-1]
		}
		if i < len(out)-1 {
			v.Next = out[i+1]
		}
	}
	return out, nil
}

// resolver returns the Markdown image resolver for project p.
func (b *builder) resolver(p *content.Project) ImageResolver {
	return func(src, alt string) (string, error) {
		if strings.Contains(src, "://") || strings.HasPrefix(src, "/") {
			return "", fmt.Errorf("image %q: only files inside the project folder are allowed", src)
		}
		file := filepath.Join(p.Dir, filepath.FromSlash(src))
		if !strings.HasPrefix(file, p.Dir+string(filepath.Separator)) {
			return "", fmt.Errorf("image %q escapes the project folder", src)
		}
		if strings.EqualFold(filepath.Ext(file), ".svg") {
			return inlineSVG(file, alt)
		}
		stem := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		im, err := b.imgs.Process(file, p.Slug+"-"+stem, imaging.Content)
		if err != nil {
			return "", err
		}
		return string(picture(im, alt, false)), nil
	}
}

// ---- Pages ------------------------------------------------------------

func (b *builder) basePage(l content.Lang, canonicalPath func(content.Lang) string) *pageData {
	s := &b.c.Site
	d := &pageData{
		Lang:     l,
		CSS:      b.css,
		FontURL:  b.manifest["/assets/fonts/plex-mono-400-latin.woff2"],
		HomeURL:  b.homeURL(l),
		UI:       map[string]string{},
		Rev:      b.opt.Now.Format("2006.01"),
		Year:     b.opt.Now.Year(),
		Profile:  b.profileView(l),
		OGImage:  b.ogImage,
		OGType:   "website",
		OGLocale: map[content.Lang]string{"tr": "tr_TR", "en": "en_US"}[l],
	}
	for k, t := range s.UI {
		d.UI[k] = t.In(l)
	}
	d.Canonical = s.BaseURL + canonicalPath(l)
	d.XDefault = s.BaseURL + canonicalPath(s.DefaultLang())
	for _, other := range s.Languages {
		d.Alternates = append(d.Alternates, alternate{other, s.BaseURL + canonicalPath(other)})
		d.Langs = append(d.Langs, langLink{
			Lang: other, Label: strings.ToUpper(string(other)),
			URL: canonicalPath(other), Current: other == l,
		})
	}
	return d
}

func (b *builder) renderHome(l content.Lang, projects []*projectView) error {
	d := b.basePage(l, b.homeURL)
	d.Title = d.Profile.Name + " · " + d.Profile.Role
	d.Description = d.Profile.Description
	d.Projects = projects
	d.OGType = "profile"
	d.JSONLD = b.personJSONLD(l)
	return b.write(b.homeURL(l), "home", d)
}

func (b *builder) renderProject(l content.Lang, p *projectView) error {
	d := b.basePage(l, func(o content.Lang) string { return b.projectURL(o, p.Slug) })
	d.Title = p.Title + " · " + d.Profile.Name
	d.Description = p.Summary
	d.OGType = "article"
	d.Project = p
	return b.write(p.URL, "project", d)
}

func (b *builder) renderNotFound() error {
	l := b.c.Site.DefaultLang()
	d := b.basePage(l, b.homeURL)
	d.Title = "404 · " + d.Profile.Name
	d.Description = d.Profile.Description
	return b.writeFile("/404.html", "notfound", d)
}

// personJSONLD describes the site owner for search engines (schema.org).
func (b *builder) personJSONLD(l content.Lang) template.JS {
	p := &b.c.Profile
	doc := map[string]any{
		"@context": "https://schema.org",
		"@type":    "Person",
		"name":     p.Name,
		"url":      b.c.Site.BaseURL + b.homeURL(l),
		"email":    "mailto:" + p.Email,
		"jobTitle": p.Role.In(l),
		"image":    b.ogImage,
		"sameAs": []string{
			"https://github.com/" + p.GitHub,
			"https://www.linkedin.com/in/" + p.LinkedIn + "/",
		},
	}
	j, _ := json.Marshal(doc)
	// json.Marshal escapes <, > and & as < etc., so the payload cannot
	// close the surrounding <script> element.
	return template.JS(j)
}

// write renders a page to <urlPath>index.html.
func (b *builder) write(urlPath, tplName string, d *pageData) error {
	b.urls = append(b.urls, urlPath)
	return b.writeFile(urlPath+"index.html", tplName, d)
}

// writeFile renders tplName with d to urlPath under the output directory and
// enforces the HTML budget.
func (b *builder) writeFile(urlPath, tplName string, d *pageData) error {
	var buf bytes.Buffer
	if err := b.tpl[tplName].ExecuteTemplate(&buf, "base", d); err != nil {
		return fmt.Errorf("render %s: %w", urlPath, err)
	}
	out := buf.Bytes()
	stat := PageStat{Path: urlPath, Bytes: len(out), Gzip: gzipSize(out)}
	b.pages = append(b.pages, stat)
	if limit := b.c.Site.HTMLBudgetBytes; stat.Bytes > limit {
		return fmt.Errorf("%s is %d bytes, over the %d byte budget in site.yaml "+
			"(shorten the page or split it, do not raise the budget lightly)", urlPath, stat.Bytes, limit)
	}
	dst := filepath.Join(b.opt.OutDir, filepath.FromSlash(urlPath))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, out, 0o644)
}

// writeExtras writes the sitemap, robots.txt and the _redirects file.
func (b *builder) writeExtras() error {
	s := &b.c.Site
	var sm strings.Builder
	sm.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sm.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	sort.Strings(b.urls)
	for _, u := range b.urls {
		fmt.Fprintf(&sm, "<url><loc>%s%s</loc></url>\n", s.BaseURL, u)
	}
	sm.WriteString("</urlset>\n")

	files := map[string]string{
		"sitemap.xml": sm.String(),
		"robots.txt":  "User-agent: *\nAllow: /\nSitemap: " + s.BaseURL + "/sitemap.xml\n",
		// "/" has no page of its own; the default language is the home.
		// Applied at Cloudflare's edge without invoking a Worker. 302
		// rather than 301 because browsers cache a 301 indefinitely, which
		// would make changing the default language later impossible.
		"_redirects": "/ /" + string(s.DefaultLang()) + "/ 302\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(b.opt.OutDir, name), []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// ---- Templates and CSS ------------------------------------------------

func parseTemplates() (map[string]*template.Template, error) {
	funcs := template.FuncMap{"join": strings.Join}
	out := map[string]*template.Template{}
	for _, name := range []string{"home", "project", "notfound"} {
		t, err := template.New(name).Funcs(funcs).ParseFS(webFS,
			"web/templates/base.gohtml", "web/templates/"+name+".gohtml")
		if err != nil {
			return nil, err
		}
		out[name] = t
	}
	return out, nil
}

var (
	cssComments  = regexp.MustCompile(`(?s)/\*.*?\*/`)
	cssSpace     = regexp.MustCompile(`\s+`)
	cssPunct     = regexp.MustCompile(`\s*([{};,>])\s*`)
	cssAssetURLs = regexp.MustCompile(`/assets/[A-Za-z0-9._/-]+`)
)

// loadCSS reads the stylesheet, rewrites asset URLs to their hashed names
// and applies a conservative minification: comments and redundant
// whitespace go, nothing is reordered or reinterpreted.
func loadCSS(manifest map[string]string) (template.CSS, error) {
	b, err := webFS.ReadFile("web/site.css")
	if err != nil {
		return "", err
	}
	s := cssComments.ReplaceAllString(string(b), "")
	s = cssSpace.ReplaceAllString(s, " ")
	s = cssPunct.ReplaceAllString(s, "$1")
	s = strings.ReplaceAll(s, ";}", "}")

	var missing []string
	s = cssAssetURLs.ReplaceAllStringFunc(s, func(u string) string {
		if h, ok := manifest[u]; ok {
			return h
		}
		missing = append(missing, u)
		return u
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("site.css references missing assets: %s", strings.Join(missing, ", "))
	}
	return template.CSS(strings.TrimSpace(s)), nil
}

// ---- Images -----------------------------------------------------------

// picture renders a processed image as <picture> with one <source> per
// modern format. eager marks images visible on first paint.
func picture(im *imaging.Image, alt string, eager bool) template.HTML {
	var b strings.Builder
	b.WriteString("<picture>")
	for _, s := range im.Sources[:len(im.Sources)-1] {
		fmt.Fprintf(&b, `<source type="%s" srcset="%s" sizes="%s">`,
			s.Format.MIME, template.HTMLEscapeString(s.SrcSet()), template.HTMLEscapeString(im.Sizes))
	}
	fb := im.Fallback()
	loading := `loading="lazy" decoding="async"`
	if eager {
		loading = `fetchpriority="high"`
	}
	fmt.Fprintf(&b, `<img src="%s" srcset="%s" sizes="%s" width="%d" height="%d" alt="%s" %s>`,
		fb.Largest().URL, template.HTMLEscapeString(fb.SrcSet()), template.HTMLEscapeString(im.Sizes),
		im.Width, im.Height, template.HTMLEscapeString(alt), loading)
	b.WriteString("</picture>")
	return template.HTML(b.String())
}

// svgPalette maps the exact colours used when drawing diagrams in
// Illustrator to CSS custom properties, so inlined diagrams follow the
// light and dark themes. Draw with these swatches; any other colour is kept
// as is. See README "Diagrams".
var svgPalette = map[string]string{
	"#000000": "var(--fg)",
	"#ffffff": "var(--bg)",
	"#808080": "var(--muted)",
	"#e6e6e6": "var(--soft)",
	"#cccccc": "var(--rule)",
	"#1f4fd1": "var(--accent)",
}

var (
	svgProlog   = regexp.MustCompile(`(?s)<\?xml.*?\?>|<!DOCTYPE.*?>|<!--.*?-->`)
	svgHex      = regexp.MustCompile(`(?i)#[0-9a-f]{6}\b|#[0-9a-f]{3}\b`)
	svgRootSize = regexp.MustCompile(`(?s)(<svg\b[^>]*?)\s(?:width|height)="[^"]*"`)
	svgRoot     = regexp.MustCompile(`<svg\b`)
)

// inlineSVG prepares an SVG file for inlining: strips the XML prolog and
// comments, removes fixed width/height so it scales with its viewBox, maps
// palette colours to theme variables and labels it for screen readers.
//
// SVG files come from the site owner's own content folder, so they are
// trusted; this is normalisation, not sanitisation.
func inlineSVG(file, alt string) (string, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	s := svgProlog.ReplaceAllString(string(b), "")
	for svgRootSize.MatchString(s) {
		s = svgRootSize.ReplaceAllString(s, "$1")
	}
	s = svgHex.ReplaceAllStringFunc(s, func(h string) string {
		h = strings.ToLower(h)
		if len(h) == 4 {
			h = "#" + strings.Repeat(h[1:2], 2) + strings.Repeat(h[2:3], 2) + strings.Repeat(h[3:4], 2)
		}
		if v, ok := svgPalette[h]; ok {
			return v
		}
		return h
	})
	label := ` role="img" aria-label="` + template.HTMLEscapeString(alt) + `"`
	s = svgRoot.ReplaceAllLiteralString(s, "<svg"+label)
	return strings.TrimSpace(s), nil
}

// ---- Files ------------------------------------------------------------

// copyTree copies every file under src to dst, preserving relative paths.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if d.Name() == ".DS_Store" || d.Name() == ".gitkeep" {
			return nil
		}
		return copyFile(p, target)
	})
}

// copyHashed copies every file under src to dst/<prefix>, inserting a short
// content hash before the extension, and returns a manifest from the
// unhashed URL to the hashed one. License texts keep their names.
func copyHashed(src, dst, prefix string) (map[string]string, error) {
	manifest := map[string]string{}
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == ".DS_Store" {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		name := rel
		if path.Ext(rel) != ".txt" {
			sum := sha256.Sum256(data)
			ext := path.Ext(rel)
			name = strings.TrimSuffix(rel, ext) + "." + hex.EncodeToString(sum[:])[:10] + ext
		}
		manifest[prefix+"/"+rel] = prefix + "/" + name
		target := filepath.Join(dst, filepath.FromSlash(strings.TrimPrefix(prefix, "/")), filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if errors.Is(err, fs.ErrNotExist) {
		return manifest, nil
	}
	return manifest, err
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// gzipSize estimates the transferred size of a page. Cloudflare serves
// Brotli to most browsers, which is a little smaller still.
func gzipSize(b []byte) int {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	w.Write(b)
	w.Close()
	return buf.Len()
}
