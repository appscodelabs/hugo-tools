package api

type Feature struct {
	Title       string `json:"title"`
	Image       Image  `json:"image"`
	Icon        Image  `json:"icon"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
}

type Image struct {
	Src string `json:"src"`
	Alt string `json:"alt"`
}

type URLRef struct {
	DomainKey string `json:"domainKey"`
	Path      string `json:"path"`
}

type ProductVersion struct {
	Branch   string `json:"branch"`
	HostDocs bool   `json:"hostDocs"`
	Show     bool   `json:"show,omitempty"`
	DocsDir  string `json:"docsDir,omitempty"` // default: "docs"
}

type Solution struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	Image       Image  `json:"image"`
	Icon        Image  `json:"icon"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
}

type Product struct {
	Key             string                `json:"key"`
	Name            string                `json:"name"`
	Tagline         string                `json:"tagline"`
	Summary         string                `json:"summary"`
	Published       bool                  `json:"published"`
	Website         URLRef                `json:"website"`
	HeroImage       Image                 `json:"heroImage"`
	Logo            Image                 `json:"logo"`
	LogoWhite       Image                 `json:"logoWhite"`
	Icon            Image                 `json:"icon"`
	RepoURL         string                `json:"repoURL,omitempty"`
	StarRepo        string                `json:"starRepo,omitempty"`
	DocRepo         string                `json:"docRepo,omitempty"`
	DatasheetFormID string                `json:"datasheetFormID,omitempty"`
	Features        []Feature             `json:"features,omitempty"`
	Solutions       []Solution            `json:"solutions,omitempty"`
	Versions        []ProductVersion      `json:"versions,omitempty"`
	LatestVersion   string                `json:"latestVersion,omitempty"`
	SocialLinks     map[string]string     `json:"socialLinks,omitempty"`
	Description     map[string]string     `json:"description,omitempty"`
	SupportLinks    map[string]string     `json:"supportLinks,omitempty"`
	StripeProductID string                `json:"stripeProductID,omitempty"`
	SubProjects     map[string]ProjectRef `json:"subProjects"`
}

type ProjectRef struct {
	Dir      string    `json:"dir"`
	Mappings []Mapping `json:"mappings"`
}

type Mapping struct {
	Versions           []string `json:"versions"`
	SubProjectVersions []string `json:"subProjectVersions"`
}