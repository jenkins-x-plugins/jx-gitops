package pipelinecatalogs

import (
	"path/filepath"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/pkg/errors"
)

// LoadPipelineCatalogs loads the pipeline catalogs and the file name for the given directory
func LoadPipelineCatalogs(dir string) (*v1alpha1.PipelineCatalog, string, error) {
	fileName := filepath.Join(dir, "extensions", v1alpha1.PipelineCatalogFileName)
	exists, err := files.FileExists(fileName)
	if err != nil {
		return nil, fileName, errors.Wrapf(err, "failed to check if file exists %s", fileName)
	}
	pipelineCatalog := &v1alpha1.PipelineCatalog{}
	if exists {
		err = yamls.LoadFile(fileName, pipelineCatalog)
		if err != nil {
			return nil, fileName, errors.Wrapf(err, "failed to load PipelineCatalog file %s", fileName)
		}
		if len(pipelineCatalog.Spec.Repositories) == 0 {
			// lets add a default repository
			pipelineCatalog.Spec.Repositories = append(pipelineCatalog.Spec.Repositories, v1alpha1.PipelineCatalogSource{
				ID:     "jx3-pipeline-catalog",
				Label:  "JX3 Pipeline Catalog",
				GitURL: "https://github.com/jstrachan/jx3-pipeline-catalog",
				GitRef: "master",
			})
		}
	}
	return pipelineCatalog, fileName, nil
}
