package chart

type Chart struct {
	Dependencies []*Dependencies `json:"dependencies"`
}

type Dependencies struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Repository string `json:"repository"`
}
