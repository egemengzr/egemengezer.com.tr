package build

import (
	"strings"
	"testing"
)

func noImages(src, alt string) (string, error) { return "<img alt=\"" + alt + "\">", nil }

func TestPipelineBlock(t *testing.T) {
	md := "```pipeline Caption\nCamera | sensor\nYOLO\n```\n"
	out, err := renderMarkdown(md, noImages)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<ol class="pipe" style="--n:3">`,
		`<li class="pipe-step" style="--i:0"><strong>Camera</strong><span>sensor</span>`,
		`<li class="pipe-link" style="--i:1" aria-hidden="true">`,
		`<li class="pipe-step" style="--i:2">`,
		`<strong>YOLO</strong></li>`,
		`<figcaption>Caption</figcaption>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	if strings.Count(out, "pipe-link") != 1 {
		t.Errorf("want exactly one connector between two steps:\n%s", out)
	}
}

func TestBarsBlock(t *testing.T) {
	md := "```bars max=60 Note\nBefore | 15 | 15 FPS\nAfter | 60 | 60 FPS | *\n```\n"
	out, err := renderMarkdown(md, noImages)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`width:25.0%`, `width:100.0%`, `class="bar bar-hl"`, `<figcaption>Note</figcaption>`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
}

func TestBarsBlockRequiresMax(t *testing.T) {
	if _, err := renderMarkdown("```bars\nA | 1 | 1\n```\n", noImages); err == nil {
		t.Fatal("want an error for a bars block without max=N")
	}
}

func TestStandaloneImageBecomesFigure(t *testing.T) {
	out, err := renderMarkdown("![Board](board.jpg \"The PCB\")\n", noImages)
	if err != nil {
		t.Fatal(err)
	}
	want := `<figure><img alt="Board"><figcaption>The PCB</figcaption></figure>`
	if !strings.Contains(out, want) || strings.Contains(out, "<p><figure>") {
		t.Errorf("got\n%s", out)
	}
}

func TestInlineImageIsRejected(t *testing.T) {
	_, err := renderMarkdown("- item\n![x](a.jpg)\n", noImages)
	if err == nil || !strings.Contains(err.Error(), "own line") {
		t.Fatalf("want an own-line error, got %v", err)
	}
}

func TestOrdinaryCodeIsEscaped(t *testing.T) {
	out, err := renderMarkdown("```c\nif (a < b) {}\n```\n", noImages)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "a &lt; b") {
		t.Errorf("code not escaped:\n%s", out)
	}
}
