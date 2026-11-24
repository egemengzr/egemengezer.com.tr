# egemengezer.com.tr

Source of my personal site: a CV and project portfolio focused on embedded
systems and embedded Linux, in Turkish and English.

It is a small static site generator written in Go. Content lives in YAML and
Markdown; the build produces plain HTML with inlined CSS, no JavaScript, and
responsive AVIF/WebP images. Cloudflare Workers serves the output as static
assets.

## Performance budget

The site is built to be readable within a second on a phone.

- **One request to first paint.** CSS is inlined, there is no JavaScript and
  no third-party origin. The only other early request is the profile photo,
  which is ~3–7 KB as AVIF.
- **Page size is enforced.** The build fails if any HTML page, including its
  CSS, exceeds `html_budget_bytes` in `content/site.yaml` (50 KB raw;
  pages are currently ~7 KB gzipped).
- **Images are sized per use.** Each image is encoded at the widths the
  layout actually renders (1x and 2x), so a phone never downloads a desktop
  image. Off-screen images load lazily.
- **Fonts.** System UI font for text; one weight of IBM Plex Mono for labels,
  split into Latin and Latin Extended so Turkish characters only download
  the second file when needed.
- **Assets are immutable.** File names contain a content hash and are cached
  for a year.

## Layout

```
content/
  site.yaml              languages, URL segments, budget, interface strings
  profile.yaml           hero, experience, skills, contact
  profile.jpg
  projects/<slug>/
    meta.yaml            the home page row: title, summary, result, tags, cover
    tr.md, en.md         the detail page body
    *.jpg, *.png, *.svg  images referenced from meta.yaml or Markdown
static/                  copied as is: favicon, _headers, cv/ (public CV PDFs)
assets/                  copied with hashed names: fonts
internal/content         loads and validates content
internal/imaging         libvips wrapper: AVIF/WebP variants, encode cache
internal/build           templates, CSS, Markdown extensions, page output
```

## Local development

Requires Go and libvips. On macOS:

```sh
brew install go vips
go run . serve        # builds, then serves http://localhost:8080
```

`go run . build` writes `dist/`. Re-run `serve` to rebuild after a change.
The first build encodes all images; later builds reuse `.cache/img`.

## Adding a project

1. Copy an existing folder under `content/projects/` and rename it. The
   folder name is the URL: lower-case words joined by hyphens.
2. Edit `meta.yaml`. `order` sets the position on the home page.
3. Write `tr.md` and `en.md`. Both are required.
4. Put images in the folder and reference them by file name. Export at
   most 1600px on the long side (JPEG ~88 quality, or PNG for
   screenshots with transparency); the build rejects files over 1 MB.
5. `go run . serve` to preview, then commit.

The build stops with a list of every problem it finds: missing
translations, unknown keys in YAML, missing images, a page over budget.

### Text fields

Any text field takes one string for every language, or one per language:

```yaml
period: "2026"
title: { tr: "İtki test standı", en: "Thrust test stand" }
```

Quote values that contain a comma. In `{ … }` style YAML a comma ends the
value; the build rejects the result rather than publishing a cut-off title.

### Markdown extensions

An image on its own line becomes a figure. The optional title is the caption:

```md
![Manufactured board](board.jpg "The stand's PCB, my first KiCad design")
```

A `pipeline` block draws a data flow, horizontal on desktop and vertical on
phones. One stage per line, `title | detail`; text after `pipeline` is the
caption:

````md
```pipeline Vision pipeline
Camera | sensor + lens + CPL filter
YOLO detection | TensorRT, FP16
```
````

A `bars` block draws a bar chart. `max=` sets full scale; `| *` highlights a
row:

````md
```bars max=60 Measured with time.perf_counter
Baseline | 13 | 12–14 FPS
Final | 57.5 | 55–60 FPS | *
```
````

### Diagrams

SVG files are inlined into the page so they follow light and dark mode.
Draw them with these exact swatches and they are mapped to theme colours:

| Swatch    | Becomes        |
| --------- | -------------- |
| `#000000` | text           |
| `#FFFFFF` | background     |
| `#808080` | secondary text |
| `#CCCCCC` | rules          |
| `#E6E6E6` | soft fill      |
| `#1F4FD1` | accent         |

Any other colour is kept as is. Paths whose id starts with `flow` (name the
layer `flow-1`, `flow-2`… in Illustrator) get a moving dashed line, unless
the visitor prefers reduced motion. Export with *Styling: Presentation
Attributes* and *Responsive* on.

### Why no Git LFS

Source images are capped at 1 MB and 1600px, so all of them together are
about 1.5 MB, and history only grows when an image actually changes. LFS
would add a separate storage and bandwidth quota (consumed by every CI
checkout) and an extra install step for anyone cloning, for no real
saving at this size.

## Deployment

CI (`.github/workflows/ci.yml`) runs on pushes to `main` and `dev` and on pull requests:
formatting, `go vet`, tests, and a full build. Pushes to `main` deploy to
Cloudflare. A failing build never deploys.

One-time setup:

1. **Cloudflare API token.** Dashboard → My Profile → API Tokens → Create
   Token → *Edit Cloudflare Workers* template. Restrict it to this account
   and the `egemengezer.com.tr` zone.
2. **GitHub secrets.** Repository → Settings → Secrets and variables →
   Actions: `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID` (from the
   Workers overview page). Then Settings → Environments → create
   `production`.
3. **DNS.** Delete any A, AAAA or CNAME record for the apex
   `egemengezer.com.tr`; the first deploy creates its own. Leave MX and TXT
   records alone: they carry e-mail.
4. **www (optional).** Rules → Redirect Rules: `www.egemengezer.com.tr/*` →
   `https://egemengezer.com.tr/${1}`, 301, with a proxied DNS record for
   `www`.

`/` redirects to `/tr/` through `static/_redirects`, generated by the build.

## Maintenance

The live site is static files: no server, database or runtime to patch.
Dependabot opens a monthly PR for Go modules and GitHub Actions; merge it
when CI is green.

## License

Code: MIT. Content (`content/`, `static/cv/`): all rights reserved. Fonts:
SIL OFL 1.1. See [LICENSE](LICENSE).
