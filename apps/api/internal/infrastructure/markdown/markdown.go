package markdown

import (
	"bytes"
	"regexp"
	"strings"

	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

var (
	spoilerRegex   = regexp.MustCompile(`\|\|(.*?)\|\|`)
	videoLinkRegex = regexp.MustCompile(`kv:<a href="(https?://[^\s]+?\.(mp4))">[^<]+</a>`)
	codeBlockRegex = regexp.MustCompile(`(?s)<pre><code class="language-(\w+)"`)

	md goldmark.Markdown
)

func init() {
	md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			mathjax.MathJax,
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
				highlighting.WithGuessLanguage(true),
			),
			&h1ToH2Extension{},
			&lazyImageExtension{},
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
}

// Render converts markdown to HTML with all custom transformations.
func Render(source string) string {
	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		return source
	}

	result := buf.String()

	// Code block wrapper:
	// <pre><code class="language-go"... → wrapped in div.kun-code-container
	result = codeBlockRegex.ReplaceAllStringFunc(result, func(match string) string {
		lang := codeBlockRegex.FindStringSubmatch(match)
		if len(lang) < 2 {
			return match
		}
		return `<div class="kun-code-container language-` + lang[1] + `">` +
			`<div class="kun-code-header">` +
			`<span class="lang">` + lang[1] + `</span>` +
			`<button class="copy" title="Copy code"></button>` +
			`</div>` +
			`<pre><code class="language-` + lang[1] + `"`
	})
	result = strings.ReplaceAll(result, "</code></pre>", "</code></pre></div>")

	// Table wrapper
	result = strings.ReplaceAll(result, "<table>", `<div class="kun-table-container"><table>`)
	result = strings.ReplaceAll(result, "</table>", `</table></div>`)

	// Spoiler: ||text|| → <span class="kun-spoiler ...">text</span>
	result = spoilerRegex.ReplaceAllString(result,
		`<span class="kun-spoiler text-transparent kun-spoiler-hidden">$1</span>`)

	// Video: kv:<a href="url.mp4">...</a> → <video>
	result = videoLinkRegex.ReplaceAllString(result,
		`<video controls loop playsinline width="100%" src="$1"></video>`)

	return result
}

// ToPlainText strips markdown syntax and returns plain text, truncated to maxLen runes.
func ToPlainText(source string, maxLen int) string {
	text := source
	text = regexp.MustCompile(`!\[.*?\]\(.*?\)`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`\[([^\]]*)\]\(.*?\)`).ReplaceAllString(text, "$1")
	text = regexp.MustCompile("[#*_~>`|]").ReplaceAllString(text, "")
	text = regexp.MustCompile(`\n{2,}`).ReplaceAllString(text, "\n")
	text = strings.TrimSpace(text)

	runes := []rune(text)
	if len(runes) > maxLen {
		return string(runes[:maxLen])
	}
	return text
}

// ──────────────────────────────────────────
// Extension: H1 → H2
// ──────────────────────────────────────────

type h1ToH2Extension struct{}

func (e *h1ToH2Extension) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&h1ToH2Renderer{}, 100),
	))
}

type h1ToH2Renderer struct{}

func (r *h1ToH2Renderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHeading, r.renderHeading)
}

func (r *h1ToH2Renderer) renderHeading(
	w util.BufWriter, source []byte, node ast.Node, entering bool,
) (ast.WalkStatus, error) {
	n := node.(*ast.Heading)
	level := n.Level
	if level == 1 {
		level = 2
	}
	tag := byte('0' + level)

	if entering {
		w.WriteString("<h")
		w.WriteByte(tag)
		if n.Attributes() != nil {
			for _, attr := range n.Attributes() {
				w.WriteByte(' ')
				w.Write(attr.Name)
				w.WriteString(`="`)
				w.Write(util.EscapeHTML(attr.Value.([]byte)))
				w.WriteByte('"')
			}
		}
		w.WriteByte('>')
	} else {
		w.WriteString("</h")
		w.WriteByte(tag)
		w.WriteString(">\n")
	}
	return ast.WalkContinue, nil
}

// ──────────────────────────────────────────
// Extension: Lazy Image
// ──────────────────────────────────────────

type lazyImageExtension struct{}

func (e *lazyImageExtension) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&lazyImageRenderer{}, 100),
	))
}

type lazyImageRenderer struct{}

func (r *lazyImageRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.renderImage)
}

func (r *lazyImageRenderer) renderImage(
	w util.BufWriter, source []byte, node ast.Node, entering bool,
) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Image)

	// Collect alt text from child text nodes
	var altBuf bytes.Buffer
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			altBuf.Write(t.Value(source))
		}
	}

	w.WriteString(`<img src="`)
	w.Write(util.EscapeHTML(n.Destination))
	w.WriteString(`" alt="`)
	w.Write(util.EscapeHTML(altBuf.Bytes()))
	if n.Title != nil {
		w.WriteString(`" title="`)
		w.Write(util.EscapeHTML(n.Title))
	}
	w.WriteString(`" loading="lazy" decoding="async" data-kun-lazy-image="true" />`)
	return ast.WalkSkipChildren, nil
}
