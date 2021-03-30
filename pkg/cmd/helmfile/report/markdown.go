package report

import (
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/releasereport"
)

// ToMarkdown converts the charts to markdown
func ToMarkdown(charts []*releasereport.NamespaceReleases) (string, error) {
	w := &strings.Builder{}

	w.WriteString("## Releases\n\n")

	w.WriteString(`
<table class="table">
  <thead>
    <tr>
      <th scope="col">Release</th>
      <th scope="col">Version</th>
      <th scope="col">Open</th>
      <th scope="col">Source</th>
    </tr>
  </thead>
  <tbody>
`)

	for _, ns := range charts {
		WriteNamespaceCharts(w, ns)
	}
	w.WriteString(`
  </tbody>
</table>

created by [Jenkins X](https://jenkins-x.io/) - see the docs on [how to configure these releases](https://jenkins-x.io/v3/develop/apps/)
`)

	return w.String(), nil
}

func WriteNamespaceCharts(w io.StringWriter, ns *releasereport.NamespaceReleases) {
	if len(ns.Releases) == 0 {
		return
	}

	w.WriteString(fmt.Sprintf(`    <tr>
		      <td colspan='4'><h3>%s</h3></td>
		    </tr>
	`, ns.Namespace))

	for _, ch := range ns.Releases {
		WriteChart(w, ch)
	}

}

func WriteChart(w io.StringWriter, ch *releasereport.ReleaseInfo) {
	description := html.EscapeString(ch.Description)

	viewLink := ""
	if ch.ApplicationURL != "" {
		viewLink = fmt.Sprintf("<a href='%s'>view</a>", ch.ApplicationURL)
	}
	sourceLink := ""
	if ch.Home != "" {
		sourceLink = fmt.Sprintf("<a href='%s'>source</a>", ch.Home)
	}

	w.WriteString(fmt.Sprintf(`    <tr>
	      <td><a href='%s' title='%s'> <img src='%s' width='24px' height='24px'> %s </a></td>
	      <td>%s</td>
	      <td>%s</td>
	      <td>%s</td>
	    </tr>
`, ch.Home, description, ch.Icon, ch.Name, ch.Version, viewLink, sourceLink))
}
