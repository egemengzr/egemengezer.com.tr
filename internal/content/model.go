package content

// Site holds site-wide configuration from content/site.yaml.
type Site struct {
	// BaseURL is the canonical origin without a trailing slash, used for
	// canonical links, hreflang alternates, Open Graph tags and the sitemap.
	BaseURL string `yaml:"base_url"`

	// Languages lists the supported languages. The first entry is the
	// default: "/" redirects to it and it is the hreflang x-default.
	Languages []Lang `yaml:"languages"`

	// ProjectsPath is the URL segment for project pages, per language
	// (for example /tr/projeler/<slug>/ and /en/projects/<slug>/).
	ProjectsPath Text `yaml:"projects_path"`

	// HTMLBudgetBytes is the maximum size, uncompressed, of any generated
	// HTML page including its inlined CSS. The build fails when a page
	// exceeds it. This keeps first paint inside a single round trip on
	// slow mobile connections; see README "Performance budget".
	HTMLBudgetBytes int `yaml:"html_budget_bytes"`

	// UI holds the fixed interface strings (button labels, headings),
	// keyed by an identifier used in the templates.
	UI map[string]Text `yaml:"ui"`
}

// DefaultLang returns the first configured language.
func (s *Site) DefaultLang() Lang { return s.Languages[0] }

// Profile is the person-level content from content/profile.yaml: the hero,
// experience, skills and contact details on the home page.
type Profile struct {
	Name     string `yaml:"name"`
	Photo    string `yaml:"photo"` // path relative to content/
	Email    string `yaml:"email"`
	GitHub   string `yaml:"github"`   // user name only
	LinkedIn string `yaml:"linkedin"` // profile slug only
	CV       Text   `yaml:"cv"`       // per language: site-absolute path of the PDF in static/

	Status      Text `yaml:"status"`      // availability line above the name
	Role        Text `yaml:"role"`        // one-line description under the name
	Summary     Text `yaml:"summary"`     // two or three sentences
	Description Text `yaml:"description"` // <meta name="description">, ~155 characters

	Facts      []Fact       `yaml:"facts"`
	Experience []Experience `yaml:"experience"`
	Skills     []Fact       `yaml:"skills"`
}

// Fact is a label/value pair rendered as a definition list entry.
type Fact struct {
	Label Text `yaml:"label"`
	Value Text `yaml:"value"`
}

// Experience is one entry in the experience section.
type Experience struct {
	Org    string `yaml:"org"`
	Period Text   `yaml:"period"`
	Role   Text   `yaml:"role"`
	Text   Text   `yaml:"text"`
}

// Project is one directory under content/projects/.
type Project struct {
	// Slug is the directory name and the last URL segment. It is not
	// configurable in meta.yaml so that the URL and the folder can never
	// drift apart.
	Slug string `yaml:"-"`

	// Dir is the absolute path of the project directory; image paths in
	// meta.yaml and the Markdown bodies are resolved against it.
	Dir string `yaml:"-"`

	// Order sorts projects on the home page, ascending.
	Order int `yaml:"order"`

	Context Text     `yaml:"context"` // where it was done: employer, team, course
	Period  Text     `yaml:"period"`
	Title   Text     `yaml:"title"`
	Summary Text     `yaml:"summary"` // one sentence for the home page row
	Result  Text     `yaml:"result"`  // the headline outcome, shown in the accent colour
	Tags    []string `yaml:"tags"`

	// Cover is an optional image relative to Dir, used as the home page
	// thumbnail and the hero image of the detail page.
	Cover string `yaml:"cover"`

	// Bars is an optional before/after chart shown on the home page row.
	Bars    []Bar   `yaml:"bars"`
	BarsMax float64 `yaml:"bars_max"`

	Links []Link `yaml:"links"`

	// Body maps each language to the raw Markdown of the detail page,
	// read from <lang>.md in Dir.
	Body map[Lang]string `yaml:"-"`
}

// Bar is one row of a horizontal bar chart.
type Bar struct {
	Label Text    `yaml:"label"`
	Value float64 `yaml:"value"`
}

// Link is an external reference such as a source code repository.
type Link struct {
	Label Text   `yaml:"label"`
	URL   string `yaml:"url"`
}

// Content is the fully loaded and validated content tree.
type Content struct {
	Root     string // absolute path of the content directory
	Site     Site
	Profile  Profile
	Projects []*Project // sorted by Order, then Slug
}
