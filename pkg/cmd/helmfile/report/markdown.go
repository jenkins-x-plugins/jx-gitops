package report

import (
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/releasereport"
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
      <th scope="col">Chart</th>
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

	_, err := w.WriteString(fmt.Sprintf(`    <tr>
		      <td colspan='5'><h3>%s</h3></td>
		    </tr>
	`, ns.Namespace))
	if err != nil {
		log.Logger().Warn(err)
	}
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

	icon := ""
	if govalidator.IsRequestURL(ch.Icon) {
		icon = fmt.Sprintf(" <img src='%s' width='24px' height='24px'>", ch.Icon)
	}
	_, err := w.WriteString(fmt.Sprintf(`    <tr>
	      <td>%s</td>
	      <td><a href='%s' title='%s'>%s %s </a></td>
	      <td>%s</td>
	      <td>%s</td>
	      <td>%s</td>
	    </tr>
`, ch.ReleaseName, ch.Home, description, icon, ch.Name, ch.Version, viewLink, sourceLink))
	if err != nil {
		log.Logger().Warn(err)
	}
}
