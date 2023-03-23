package report

import (
	"bytes"
	"os"
	"text/template"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-api/v4/pkg/cloud"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
)

func getLogURL(requirements *jxcore.RequirementsConfig, ns, containerName string) string {
	c := &requirements.Cluster
	logsURL, envExists := os.LookupEnv("JX_LOGS_URL")
	if envExists {
		t, err := template.New("logsURL").Parse(logsURL)
		if err != nil {
			log.Logger().Warnf("failed to parse environment variable JX_LOGS_URL (%s) as template: %v", logsURL, err)
		} else {
			data := map[string]interface{}{
				"requirements": requirements,
				"ns":           ns,
				"container":    containerName,
			}
			var tpl bytes.Buffer
			err := t.Execute(&tpl, data)
			if err != nil {
				log.Logger().Warnf("failed to execute template %s with values: %v\nerror: %v", logsURL, data, err)
			} else {
				return tpl.String()
			}
		}
	}
	if c.Provider == cloud.GKE {
		return logsURLForGCP(c.ProjectID, c.ClusterName, ns, containerName)
	}
	return ""
}

// logsURLForGCP generates the URL for a container logs URL
func logsURLForGCP(projectName, clusterName, ns, containerName string) string {
	if projectName != "" && clusterName != "" && containerName != "" {
		return `https://console.cloud.google.com/logs/viewer?authuser=1&project=` + projectName + `&minLogLevel=0&expandAll=false&customFacets=&limitCustomFacetWidth=true&interval=PT1H&resource=k8s_container%2Fcluster_name%2F` + clusterName + `%2Fnamespace_name%2F` + ns + `%2Fcontainer_name%2F` + containerName + `&dateRangeUnbound=both`
	}
	return ""
}
