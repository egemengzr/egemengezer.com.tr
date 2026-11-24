package build

import (
	"bytes"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// This file extends Markdown with the few constructs the project pages need,
// so that authors write plain text and never hand-write HTML:
//
//   - A paragraph that holds only an image becomes a <figure>. Raster images
//     are emitted as responsive <picture> elements; SVG files are inlined so
//     they inherit the page's colours (dark mode) and animations. The image
//     title, ![alt](src "title"), becomes the caption.
//
//   - A fenced block with the info string "pipeline" draws a left-to-right
//     data flow (top-to-bottom on phones). One stage per line, title and
//     detail separated by "|". Text after "pipeline" is the caption:
//
//     ```pipeline Vision pipeline
//     Camera | sensor + lens + CPL filter
//     YOLO detection | TensorRT, FP16
//     ```
//
//   - A fenced block with the info string "bars" draws a horizontal bar
//     chart. "max=N" sets the full-scale value; one row per line as
//     label | value | shown text, with a trailing "| *" marking a row to
//     highlight in the accent colour.
//
// Raw HTML in Markdown is not enabled: all markup comes from this file and
// the templates, which keeps pages consistent and the CSS small.

// ImageResolver turns an image reference from Markdown into figure HTML.
// It is supplied per page because paths are relative to the project folder.
type ImageResolver func(src, alt string) (string, error)

// renderMarkdown converts one page body to HTML.
func renderMarkdown(source string, resolve ImageResolver) (string, error) {
	r := &extRenderer{resolve: resolve}
	md := goldmark.New(
		goldmark.WithExtensions(extension.Table, extension.Strikethrough),
		goldmark.WithParserOptions(
			parser.WithAttribute(),
			parser.WithASTTransformers(util.Prioritized(figureTransformer{}, 100)),
		),
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(util.Prioritized(r, 100)),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		return "", err
	}
	if r.err != nil {
		return "", r.err
	}
	return buf.String(), nil
}

// kindFigure is the AST node kind for a stand-alone image.
var kindFigure = ast.NewNodeKind("Figure")

// figureNode replaces a paragraph whose only child is an image.
type figureNode struct {
	ast.BaseBlock
	Src, Alt, Caption string
}

func (n *figureNode) Kind() ast.NodeKind { return kindFigure }
func (n *figureNode) Dump(src []byte, level int) {
	ast.DumpHelper(n, src, level, map[string]string{"Src": n.Src}, nil)
}

// figureTransformer lifts stand-alone images out of their paragraphs,
// because a <figure> may not appear inside a <p>.
type figureTransformer struct{}

func (figureTransformer) Transform(doc *ast.Document, reader text.Reader, _ parser.Context) {
	src := reader.Source()
	var targets []*ast.Paragraph
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if p, ok := n.(*ast.Paragraph); ok && entering && p.ChildCount() == 1 {
			if _, ok := p.FirstChild().(*ast.Image); ok {
				targets = append(targets, p)
			}
		}
		return ast.WalkContinue, nil
	})
	for _, p := range targets {
		img := p.FirstChild().(*ast.Image)
		fig := &figureNode{
			Src:     string(img.Destination),
			Alt:     plainText(img, src),
			Caption: string(img.Title),
		}
		p.Parent().ReplaceChild(p.Parent(), p, fig)
	}
}

// plainText concatenates the text segments under n (the image alt text).
func plainText(n ast.Node, src []byte) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			b.Write(t.Segment.Value(src))
		} else {
			b.WriteString(plainText(c, src))
		}
	}
	return b.String()
}

// extRenderer renders figures and the custom fenced blocks, and defers to
// goldmark's default renderer for everything else.
type extRenderer struct {
	resolve ImageResolver
	err     error // first resolver error; goldmark has no error channel here
}

func (r *extRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindFigure, r.renderFigure)
	reg.Register(ast.KindFencedCodeBlock, r.renderFenced)
	reg.Register(ast.KindImage, r.renderInlineImage)
}

// renderInlineImage rejects images that are not alone in their paragraph,
// for example one that Markdown's lazy continuation pulled into a list item.
// Such an image would bypass the resolver and ship with a broken URL.
func (r *extRenderer) renderInlineImage(_ util.BufWriter, _ []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering && r.err == nil {
		r.err = fmt.Errorf("image %q must be on its own line with a blank line before and after",
			n.(*ast.Image).Destination)
	}
	return ast.WalkSkipChildren, nil
}

func (r *extRenderer) renderFigure(w util.BufWriter, _ []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	f := n.(*figureNode)
	body, err := r.resolve(f.Src, f.Alt)
	if err != nil && r.err == nil {
		r.err = err
	}
	w.WriteString("<figure>")
	w.WriteString(body)
	if f.Caption != "" {
		w.WriteString("<figcaption>" + html.EscapeString(f.Caption) + "</figcaption>")
	}
	w.WriteString("</figure>\n")
	return ast.WalkSkipChildren, nil
}

func (r *extRenderer) renderFenced(w util.BufWriter, src []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	cb := n.(*ast.FencedCodeBlock)
	lang := string(cb.Language(src))
	if lang != "pipeline" && lang != "bars" {
		// Ordinary code block: escape and emit as <pre><code>.
		if entering {
			w.WriteString("<pre><code>")
			for i := 0; i < cb.Lines().Len(); i++ {
				line := cb.Lines().At(i)
				w.WriteString(html.EscapeString(string(line.Value(src))))
			}
			w.WriteString("</code></pre>\n")
		}
		return ast.WalkSkipChildren, nil
	}
	if !entering {
		return ast.WalkContinue, nil
	}

	info := ""
	if cb.Info != nil {
		info = strings.TrimSpace(strings.TrimPrefix(string(cb.Info.Segment.Value(src)), lang))
	}
	var lines []string
	for i := 0; i < cb.Lines().Len(); i++ {
		seg := cb.Lines().At(i)
		if l := strings.TrimSpace(string(seg.Value(src))); l != "" {
			lines = append(lines, l)
		}
	}

	var out string
	var err error
	if lang == "pipeline" {
		out = pipelineHTML(info, lines)
	} else {
		out, err = barsHTML(info, lines)
	}
	if err != nil && r.err == nil {
		r.err = err
	}
	w.WriteString(out)
	return ast.WalkSkipChildren, nil
}

// pipelineHTML renders the pipeline block. Connectors are decorative and
// hidden from assistive technology; the ordered list carries the meaning.
//
// Every item carries its position as --i and the list its length as --n,
// so the stylesheet can run one pulse through the stages in order (see
// "Pipeline" in site.css) without any per-diagram CSS.
func pipelineHTML(caption string, lines []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<figure class="pipe-fig"><ol class="pipe" style="--n:%d">`, 2*len(lines)-1)
	for i, l := range lines {
		title, detail, _ := strings.Cut(l, "|")
		if i > 0 {
			fmt.Fprintf(&b, `<li class="pipe-link" style="--i:%d" aria-hidden="true"></li>`, 2*i-1)
		}
		fmt.Fprintf(&b, `<li class="pipe-step" style="--i:%d"><strong>`, 2*i)
		b.WriteString(html.EscapeString(strings.TrimSpace(title)) + `</strong>`)
		if d := strings.TrimSpace(detail); d != "" {
			b.WriteString(`<span>` + html.EscapeString(d) + `</span>`)
		}
		b.WriteString(`</li>`)
	}
	b.WriteString(`</ol>`)
	if caption != "" {
		b.WriteString(`<figcaption>` + html.EscapeString(caption) + `</figcaption>`)
	}
	b.WriteString("</figure>\n")
	return b.String()
}

// barsHTML renders the bars block, scaling each bar against max=N.
func barsHTML(info string, lines []string) (string, error) {
	max := 0.0
	caption := info
	if strings.HasPrefix(info, "max=") {
		head, rest, _ := strings.Cut(info, " ")
		v, err := strconv.ParseFloat(strings.TrimPrefix(head, "max="), 64)
		if err != nil || v <= 0 {
			return "", fmt.Errorf("bars block: invalid %q", head)
		}
		max, caption = v, strings.TrimSpace(rest)
	}
	if max == 0 {
		return "", fmt.Errorf("bars block: missing max=N in %q", info)
	}

	var b strings.Builder
	b.WriteString(`<figure class="bars-fig"><dl class="bars">`)
	for _, l := range lines {
		cols := strings.Split(l, "|")
		for i := range cols {
			cols[i] = strings.TrimSpace(cols[i])
		}
		if len(cols) < 3 {
			return "", fmt.Errorf("bars block: want label | value | text, got %q", l)
		}
		v, err := strconv.ParseFloat(cols[1], 64)
		if err != nil {
			return "", fmt.Errorf("bars block: value %q: %w", cols[1], err)
		}
		cls := "bar"
		if len(cols) > 3 && cols[3] == "*" {
			cls += " bar-hl"
		}
		pct := clampPct(v / max * 100)
		fmt.Fprintf(&b, `<div class="%s"><dt>%s</dt><dd><span class="bar-val">%s</span><span class="bar-track"><span class="bar-fill" style="width:%.1f%%"></span></span></dd></div>`,
			cls, html.EscapeString(cols[0]), html.EscapeString(cols[2]), pct)
	}
	b.WriteString(`</dl>`)
	if caption != "" {
		b.WriteString(`<figcaption>` + html.EscapeString(caption) + `</figcaption>`)
	}
	b.WriteString("</figure>\n")
	return b.String(), nil
}

func clampPct(p float64) float64 {
	switch {
	case p < 0:
		return 0
	case p > 100:
		return 100
	}
	return p
}
