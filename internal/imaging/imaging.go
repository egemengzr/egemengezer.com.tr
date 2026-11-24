// Package imaging turns source images from the content tree into the
// responsive, modern-format variants the site serves.
//
// Every raster image is re-encoded as AVIF and WebP at a small set of widths
// chosen per use (a preset), and the page offers them through <picture> and
// srcset so that a phone never downloads a desktop-sized file. Encoding is
// delegated to the libvips command line tools, which are fast, memory-frugal
// and available on both macOS (Homebrew) and the Ubuntu CI runners.
//
// Output file names embed a hash of the source bytes, so variants can be
// cached by browsers forever (see static/_headers) and a changed source
// automatically gets a new URL.
package imaging

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// Format is an output encoding together with its libvips save options.
type Format struct {
	Ext  string // file extension without the dot
	MIME string // value for <source type="…">
	Opts string // libvips saver options, appended as file.ext[Opts]
}

// The encodings the site uses. Qualities were chosen by eye on the
// portfolio photographs: AVIF at Q55 and WebP at Q78 are visually
// indistinguishable from the source at the rendered sizes. JPEG exists only
// for Open Graph previews, because some link unfurlers ignore WebP/AVIF.
var (
	AVIF = Format{Ext: "avif", MIME: "image/avif", Opts: "Q=55,effort=4,strip"}
	WebP = Format{Ext: "webp", MIME: "image/webp", Opts: "Q=78,effort=5,strip"}
	JPEG = Format{Ext: "jpg", MIME: "image/jpeg", Opts: "Q=82,strip,optimize_coding"}
)

// Preset describes how one kind of image is used on the page.
type Preset struct {
	Name    string
	Widths  []int    // candidate output widths in pixels, ascending
	Formats []Format // in order of preference; the last one is the <img> fallback
	Sizes   string   // the sizes attribute: rendered width per viewport
}

// Presets used by the templates. Widths cover 1x and 2x displays for the
// rendered CSS sizes in internal/build/web/site.css; keep them in step.
var (
	// Portrait is the profile photo: 96px wide on phones, 232px on desktop.
	Portrait = Preset{Name: "portrait", Widths: []int{232, 464}, Formats: []Format{AVIF, WebP},
		Sizes: "(min-width: 760px) 232px, 96px"}
	// Thumb is a project row thumbnail: 88px on phones, 184px on desktop.
	Thumb = Preset{Name: "thumb", Widths: []int{184, 368}, Formats: []Format{AVIF, WebP},
		Sizes: "(min-width: 760px) 184px, 88px"}
	// Content is an image inside an article column of at most 720px.
	Content = Preset{Name: "content", Widths: []int{480, 960, 1440}, Formats: []Format{AVIF, WebP},
		Sizes: "(min-width: 760px) 720px, calc(100vw - 40px)"}
	// OpenGraph is the link preview image for social sites and messengers.
	OpenGraph = Preset{Name: "og", Widths: []int{600}, Formats: []Format{JPEG}}
)

// Variant is one encoded file.
type Variant struct {
	URL   string
	Width int
}

// Source groups the variants of one format.
type Source struct {
	Format   Format
	Variants []Variant
}

// SrcSet renders the variants as a srcset attribute value.
func (s Source) SrcSet() string {
	parts := make([]string, len(s.Variants))
	for i, v := range s.Variants {
		parts[i] = v.URL + " " + strconv.Itoa(v.Width) + "w"
	}
	return strings.Join(parts, ", ")
}

// Largest returns the widest variant.
func (s Source) Largest() Variant { return s.Variants[len(s.Variants)-1] }

// Image is a processed source image, ready for markup.
type Image struct {
	Sources []Source // one per preset format, in preference order
	Sizes   string
	// Width and Height are the intrinsic size of the largest variant. They
	// go on the <img> so the browser reserves space before the file
	// arrives and the layout does not shift.
	Width, Height int
}

// Fallback returns the source used for the plain <img> element.
func (im *Image) Fallback() Source { return im.Sources[len(im.Sources)-1] }

// Processor encodes images into an output directory, reusing previously
// encoded files from a cache directory across builds.
//
// A Processor is safe for concurrent use.
type Processor struct {
	outDir    string // site output root; files go to outDir/urlPrefix
	urlPrefix string // URL path of the image directory, e.g. "/assets/img"
	cacheDir  string // persistent encode cache, e.g. ".cache/img"

	sem chan struct{} // bounds concurrent libvips processes

	mu   sync.Mutex
	done map[string]*Image // memoises Process by source hash and preset
}

// NewProcessor returns a Processor writing under outDir and caching under
// cacheDir. It fails early, with installation hints, when libvips is not
// available, rather than halfway through a build.
func NewProcessor(outDir, urlPrefix, cacheDir string) (*Processor, error) {
	for _, tool := range []string{"vips", "vipsheader"} {
		if _, err := exec.LookPath(tool); err != nil {
			return nil, fmt.Errorf("%s not found on PATH: install libvips "+
				"(macOS: brew install vips; Ubuntu: apt install libvips-tools libheif-plugin-aomenc)", tool)
		}
	}
	dest := filepath.Join(outDir, filepath.FromSlash(strings.TrimPrefix(urlPrefix, "/")))
	for _, d := range []string{dest, cacheDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	return &Processor{
		outDir:    outDir,
		urlPrefix: urlPrefix,
		cacheDir:  cacheDir,
		sem:       make(chan struct{}, runtime.NumCPU()),
		done:      map[string]*Image{},
	}, nil
}

// Process encodes src for preset and returns the resulting Image. name is a
// human-readable stem for the output files (for example "vision-cover");
// the content hash is appended to it, so it only needs to be descriptive.
func (p *Processor) Process(src, name string, preset Preset) (*Image, error) {
	hash, err := fileHash(src)
	if err != nil {
		return nil, err
	}
	key := hash + "/" + preset.Name
	p.mu.Lock()
	if im, ok := p.done[key]; ok {
		p.mu.Unlock()
		return im, nil
	}
	p.mu.Unlock()

	w, h, err := dimensions(src)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", src, err)
	}
	widths := targetWidths(preset.Widths, w)

	im := &Image{Sizes: preset.Sizes}
	var (
		wg   sync.WaitGroup
		errs = make([]error, len(preset.Formats)*len(widths))
	)
	for fi, f := range preset.Formats {
		s := Source{Format: f, Variants: make([]Variant, len(widths))}
		for wi, tw := range widths {
			file := fmt.Sprintf("%s-%d.%s.%s", name, tw, hash[:10], f.Ext)
			s.Variants[wi] = Variant{URL: p.urlPrefix + "/" + file, Width: tw}

			wg.Add(1)
			go func(idx int, file string, tw int, f Format) {
				defer wg.Done()
				p.sem <- struct{}{}
				defer func() { <-p.sem }()
				errs[idx] = p.encode(src, file, tw, f)
			}(fi*len(widths)+wi, file, tw, f)
		}
		im.Sources = append(im.Sources, s)
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("%s: %w", src, err)
	}

	largest := widths[len(widths)-1]
	im.Width = largest
	im.Height = int(math.Round(float64(h) * float64(largest) / float64(w)))

	p.mu.Lock()
	p.done[key] = im
	p.mu.Unlock()
	return im, nil
}

// encode produces one variant, from the cache when possible, and copies it
// into the output directory.
func (p *Processor) encode(src, file string, width int, f Format) error {
	cached := filepath.Join(p.cacheDir, file)
	if _, err := os.Stat(cached); err != nil {
		// Encode to a temporary name and rename, so an interrupted build
		// never leaves a truncated file in the cache.
		tmp := cached + ".tmp." + f.Ext
		cmd := exec.Command("vips", "thumbnail", src, tmp+"["+f.Opts+"]", strconv.Itoa(width),
			// A very large height makes the width the only constraint;
			// --size down guarantees we never upscale.
			"--height", "100000", "--size", "down")
		if out, err := cmd.CombinedOutput(); err != nil {
			os.Remove(tmp)
			return fmt.Errorf("vips thumbnail: %v: %s", err, strings.TrimSpace(string(out)))
		}
		if err := os.Rename(tmp, cached); err != nil {
			return err
		}
	}
	dest := filepath.Join(p.outDir, filepath.FromSlash(strings.TrimPrefix(p.urlPrefix, "/")), file)
	return copyFile(cached, dest)
}

// targetWidths drops widths larger than the source, since upscaling only
// adds bytes, and keeps the source width itself when it falls between two
// candidates so that the sharpest available version is still offered.
func targetWidths(candidates []int, srcWidth int) []int {
	var out []int
	for _, w := range candidates {
		if w <= srcWidth {
			out = append(out, w)
		}
	}
	if len(out) == 0 || (out[len(out)-1] < srcWidth && len(out) < len(candidates)) {
		out = append(out, srcWidth)
	}
	return out
}

// dimensions reads the pixel size of an image via vipsheader, which
// understands every format libvips can load, after EXIF auto-rotation.
func dimensions(path string) (w, h int, err error) {
	out, err := exec.Command("vipsheader", "-f", "width", path).Output()
	if err != nil {
		return 0, 0, fmt.Errorf("vipsheader: %w", err)
	}
	w, err = strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, 0, err
	}
	out, err = exec.Command("vipsheader", "-f", "height", path).Output()
	if err != nil {
		return 0, 0, fmt.Errorf("vipsheader: %w", err)
	}
	h, err = strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, 0, err
	}
	if o, err := exec.Command("vipsheader", "-f", "orientation", path).Output(); err == nil {
		// EXIF orientations 5–8 rotate by 90°, which swaps the axes that
		// vips thumbnail will output.
		if n, _ := strconv.Atoi(strings.TrimSpace(string(o))); n >= 5 && n <= 8 {
			w, h = h, w
		}
	}
	return w, h, nil
}

// fileHash returns the hex SHA-256 of a file's contents.
func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// copyFile copies src to dst, creating parent directories as needed.
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
