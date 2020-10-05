package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/versionstream"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

// main converts the 2.x version stream files to 3.x format
func main() {
	dir := "versionStream"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	err := convert(dir)
	if err != nil {
		log.Logger().Fatal(err.Error())
		return
	}
	log.Logger().Infof("completed")
}

func convert(dir string) error {
	log.Logger().Infof("converting version stream files from dir %s", dir)

	chartsDir := filepath.Join(dir, "charts")
	appsDir := filepath.Join(dir, "apps")

	exists, err := files.DirExists(chartsDir)
	if err != nil {
		return errors.Wrapf(err, "failed to ")
	}
	if !exists {
		return errors.Errorf("no directory %s", chartsDir)
	}

	err = filepath.Walk(chartsDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || !strings.HasSuffix(path, ".yml") {
			return nil
		}

		chartDir := strings.TrimSuffix(path, ".yml")
		rel, err := filepath.Rel(chartsDir, chartDir)

		versions, err := versionstream.LoadStableVersionFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to load versions file %s", path)
		}

		defaultsFile := filepath.Join(appsDir, rel, "defaults.yml")
		exists, err = files.FileExists(defaultsFile)
		if err != nil {
			return errors.Wrapf(err, "failed to check file exists %s", defaultsFile)
		}
		if exists {
			defaultConfig := &AppDefaultsConfig{}
			err = yamls.LoadFile(defaultsFile, &defaultConfig)
			if err != nil {
				return errors.Wrapf(err, "failed to load defaults file %s", defaultsFile)
			}
			if defaultConfig.Namespace != "" {
				log.Logger().Infof("loaded defaults file %s and found namespace %s", defaultsFile, defaultConfig.Namespace)

				versions.Namespace = defaultConfig.Namespace
			}
		}

		err = os.MkdirAll(chartDir, files.DefaultDirWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to make chart dir %s", chartDir)
		}

		fileName := filepath.Join(chartDir, "defaults.yaml")
		err = versionstream.SaveStableVersionFile(fileName, versions)
		if err != nil {
			return errors.Wrapf(err, "failed to save new versions file %s", fileName)
		}
		log.Logger().Infof("saved file %s", fileName)
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to process charts")
	}
	return nil
}

// AppDefaultsConfig contains optional defaults for apps installed via helmfile / helm 3 which are
// typically discovered from the Version Stream
type AppDefaultsConfig struct {
	// Namespace the default namespace to install this app into
	Namespace string `json:"namespace,omitempty"`
	// Phase the boot phase this app should be installed in. Leave blank if you are not sure.
	// things like ingress controllers are in 'system' and most other things default to the 'apps' phase
	Phase string `json:"phase,omitempty"`
}
