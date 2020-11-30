package resolve

import (
	"io/ioutil"
	"net/http"
	"strings"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/httphelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

const (
	// GKEMetadataEndpoint default Google metadata endpoint.
	// See https://cloud.google.com/compute/docs/storing-retrieving-metadata#querying
	GKEMetadataEndpoint = "http://metadata.google.internal/computeMetadata/v1/"

	// paths in the REST API...

	// GKEPathProjectID metadata endpoint path to the project ID string
	GKEPathProjectID = "project/project-id"

	// GKEPathProjectNumber metadata endpoint path to the project number
	GKEPathProjectNumber = "project/numeric-project-id"

	// GKEPathClusterName metadata endpoint path to the cluster name
	GKEPathClusterName = "instance/attributes/cluster-name"

	// GKEPathClusterLocation metadata endpoint path to the cluster location
	GKEPathClusterLocation = "instance/attributes/cluster-location"
)

// GKEConfig the GKE specific configuration
type GKEConfig struct {
	MetadataEndpoint string
}

// ResolveGKE resolves any missing GKE metadata
func (o *Options) ResolveGKE() error {
	cluster := &o.requirements.Spec.Cluster
	if cluster.GKEConfig == nil {
		cluster.GKEConfig = &jxcore.GKEConfig{}
	}
	projectID := cluster.ProjectID
	clusterName := cluster.ClusterName
	projectNumber := cluster.GKEConfig.ProjectNumber
	region := cluster.Region
	zone := cluster.Zone
	location := region
	if location == "" {
		location = zone
	}

	if projectID != "" && clusterName != "" && projectNumber != "" && (zone != "" || region != "") {
		o.logGKEMetadata()
		return nil
	}

	if !o.NoInClusterCheck && !IsInCluster() {
		log.Logger().Warnf("cannot default GKE metadata as this command is not running inside the cluster")
		return nil
	}

	log.Logger().Infof("resolving missing GKE project and cluster metadata from endpoint %s", o.getGKEMetadataEndpoint())
	var err error
	if projectID == "" {
		cluster.ProjectID, err = o.getGKEMetadata(GKEPathProjectID)
		if err != nil {
			return err
		}
	}
	if projectNumber == "" {
		cluster.GKEConfig.ProjectNumber, err = o.getGKEMetadata(GKEPathProjectNumber)
		if err != nil {
			return err
		}
	}
	if clusterName == "" {
		cluster.ClusterName, err = o.getGKEMetadata(GKEPathClusterName)
		if err != nil {
			return err
		}
	}
	if location == "" {
		location, err := o.getGKEMetadata(GKEPathClusterLocation)
		if err != nil {
			return err
		}

		// TODO lets assume a zone for now
		cluster.Zone = location
	}

	err = o.requirements.SaveConfig(o.requirementsFileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save modified requirements file: %s", o.requirementsFileName)
	}

	log.Logger().Infof("resolved GKE project and cluster metadata and modified file %s", termcolor.ColorInfo(o.requirementsFileName))
	o.logGKEMetadata()

	if o.NoCommit {
		return nil
	}
	gitter := o.GitClient()
	_, err = gitter.Command(o.Dir, "add", "*")
	if err != nil {
		return errors.Wrapf(err, "failed to add to git")
	}
	err = gitclient.CommitIfChanges(gitter, o.Dir, "chore: default GKE project, cluster and location metadata")
	if err != nil {
		return errors.Wrapf(err, "failed to git commit the changes to the GKE project, cluster and location")
	}
	return nil
}

func (o *Options) getGKEMetadata(path string) (string, error) {
	ep := o.getGKEMetadataEndpoint()
	u := stringhelpers.UrlJoin(ep, path)

	client := httphelpers.GetClient()
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create http request for %s", u)
	}
	req.Header.Add("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		if resp != nil {
			return "", errors.Wrapf(err, "failed to GET endpoint %s with status %s", u, resp.Status)
		}
		return "", errors.Wrapf(err, "failed to GET endpoint %s", u)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read response from %s", u)
	}
	return strings.TrimSpace(string(body)), nil
}

func (o *Options) getGKEMetadataEndpoint() string {
	if o.GKEConfig.MetadataEndpoint == "" {
		o.GKEConfig.MetadataEndpoint = GKEMetadataEndpoint
	}
	return o.GKEConfig.MetadataEndpoint
}

func (o *Options) logGKEMetadata() {
	cluster := &o.requirements.Spec.Cluster
	info := termcolor.ColorInfo

	log.Logger().Infof("GKE project: %s", info(cluster.ProjectID))
	log.Logger().Infof("project number: %s", info(cluster.GKEConfig.ProjectNumber))
	log.Logger().Infof("cluster name: %s", info(cluster.ClusterName))
	if cluster.Region != "" {
		log.Logger().Infof("region: %s", info(cluster.Region))
	} else {
		log.Logger().Infof("zone: %s", info(cluster.Zone))
	}
}
