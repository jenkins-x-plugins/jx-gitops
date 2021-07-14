package chart

type Chart struct {
	Dependencies []*ChartDependencies `json:"dependencies"`
}

type ChartDependencies struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Repository string `json:"repository"`
}
