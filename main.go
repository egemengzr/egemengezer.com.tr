// Command site builds and previews egemengezer.com.tr.
//
// Usage:
//
//	go run . build            # render content/ into dist/
//	go run . serve            # build, then serve dist/ on http://localhost:8080
//	go run . serve -addr :9000
//
// The build is the same locally and in CI; CI additionally deploys dist/ to
// Cloudflare Workers when main changes (see .github/workflows/ci.yml).
package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"egemengezer.com.tr/site/internal/build"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: site build | site serve [-addr :8080]")
	}
	cmd, rest := args[0], args[1:]

	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	out := fs.String("out", "dist", "output directory")
	addr := fs.String("addr", "localhost:8080", "serve: listen address")
	if err := fs.Parse(rest); err != nil {
		return err
	}

	opt := build.Options{
		ContentDir: "content",
		StaticDir:  "static",
		AssetsDir:  "assets",
		OutDir:     *out,
		CacheDir:   filepath.Join(".cache", "img"),
		Now:        now(),
	}

	switch cmd {
	case "build":
		return buildAndReport(opt)
	case "serve":
		if err := buildAndReport(opt); err != nil {
			return err
		}
		fmt.Printf("\nserving %s on http://%s  (Ctrl-C to stop; re-run to rebuild)\n", *out, *addr)
		return http.ListenAndServe(*addr, previewHandler(*out))
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

// now returns the build time in Istanbul, which sets the "Rev." label and
// the copyright year. SOURCE_DATE_EPOCH, when set, makes builds
// reproducible (https://reproducible-builds.org/specs/source-date-epoch/).
func now() time.Time {
	loc, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		loc = time.FixedZone("TRT", 3*60*60)
	}
	if s := os.Getenv("SOURCE_DATE_EPOCH"); s != "" {
		var sec int64
		if _, err := fmt.Sscan(s, &sec); err == nil {
			return time.Unix(sec, 0).In(loc)
		}
	}
	return time.Now().In(loc)
}

func buildAndReport(opt build.Options) error {
	start := time.Now()
	rep, err := build.Build(opt)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(w, "page\tbytes\tgzip\t")
	for _, p := range rep.Pages {
		fmt.Fprintf(w, "%s\t%d\t%d\t\n", p.Path, p.Bytes, p.Gzip)
	}
	w.Flush()
	fmt.Printf("built %d pages in %s\n", len(rep.Pages), time.Since(start).Round(time.Millisecond))
	return nil
}

// previewHandler serves the output directory roughly the way Cloudflare
// does: "/" redirects to the default language (read from _redirects),
// directories serve index.html and unknown paths get 404.html.
func previewHandler(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			if b, err := os.ReadFile(filepath.Join(dir, "_redirects")); err == nil {
				if f := strings.Fields(string(b)); len(f) >= 2 {
					http.Redirect(w, r, f[1], http.StatusFound)
					return
				}
			}
		}
		p := filepath.Join(dir, filepath.FromSlash(path.Clean(r.URL.Path)))
		if st, err := os.Stat(p); err == nil {
			if !st.IsDir() {
				files.ServeHTTP(w, r)
				return
			}
			if _, err := os.Stat(filepath.Join(p, "index.html")); err == nil {
				files.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		if b, err := os.ReadFile(filepath.Join(dir, "404.html")); err == nil {
			w.Write(b)
		}
	})
}
