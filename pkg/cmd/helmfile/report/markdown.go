package report

import (
	"fmt"
	"html"
	"io"
	"strings"
)

// ToMarkdown converts the charts to markdown
func ToMarkdown(charts []*NamespaceCharts) (string, error) {
	buf := &strings.Builder{}

	buf.WriteString("# Deployments\n\n")

	for _, ns := range charts {
		WriteNamespaceCharts(buf, ns)
	}
	return buf.String(), nil
}

func WriteNamespaceCharts(w io.StringWriter, ns *NamespaceCharts) {
	w.WriteString("\n## " + ns.Namespace + "\n\n")

	for _, ch := range ns.Charts {
		WriteChart(w, ch)
	}
}

func WriteChart(w io.StringWriter, ch *ChartInfo) {
	description := html.EscapeString(ch.Description)
	w.WriteString(fmt.Sprintf("* <a href='%s' title='%s'> <img src='%s' width='24px' height='24px'> %s </a> %s\n", ch.Home, description, ch.Icon, ch.Name, ch.Version))
}
