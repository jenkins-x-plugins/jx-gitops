package helmhelpers

const (
	// JX3HelmRepository the default "jx3" helm repository
	JX3HelmRepository = "https://jenkins-x-charts.github.io/repo"

	// VersionLabel the label on helmfile releases to avoid overriding the version
	VersionLabel = "version.jenkins-x.io"

	// ValuesLabel the label on helmfile releases to avoid overriding the values YAML
	ValuesLabel = "values.jenkins-x.io"

	// LockLabelValue the value of the VersionLabel or ValuesLabel to lock the values and not apply version stream values
	LockLabelValue = "lock"
)
