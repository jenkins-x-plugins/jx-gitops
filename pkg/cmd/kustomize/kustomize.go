package kustomize

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/common"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	splitLong = templates.LongDesc(`
		Generates a kustomize layout by comparing a source and target directories.

If you are using kpt to consume templates and you make lots of modifications and hit merge/upgrade issues this command lets you reverse engineer kustomize overlays from the changes you have made the to resources. 

`)

	splitExample = templates.Examples(`
		# reverse engineer kustomize overlays by comparing the source to the current target
		%s kustomize --source src/base --target config-root --output src/overlays/default
	`)

	// mandatoryFields fields we should not remove when creating a diff
	mandatoryFields = []string{"apiVersion", "kind", "metadata.name"}
)

// Options the options for the command
type Options struct {
	SourceDir string
	TargetDir string
	OutputDir string
}

// NewCmdKustomize creates a command object for the command
func NewCmdKustomize() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "kustomize",
		Short:   "Generates a kustomize layout by comparing a source and target directories",
		Long:    splitLong,
		Example: fmt.Sprintf(splitExample, common.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.SourceDir, "source", "s", ".", "the directory to recursively look for the source *.yaml or *.yml files")
	cmd.Flags().StringVarP(&o.TargetDir, "target", "t", "", "the directory to recursively look for the target *.yaml or *.yml files")
	cmd.Flags().StringVarP(&o.OutputDir, "output", "o", "", "the output directory to store the overlays")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	target := o.TargetDir
	if target == "" {
		return util.MissingOption("target")
	}
	dir := o.SourceDir

	var err error
	if o.OutputDir != "" {
		err = os.MkdirAll(o.OutputDir, util.DefaultWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to create dir %s", o.OutputDir)
		}
	} else {
		o.OutputDir, err = ioutil.TempDir("", "")
		if err != nil {
			return errors.Wrapf(err, "failed to create a temp dir")
		}
	}
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return errors.Wrapf(err, "failed to calculate the relative directory of %s", path)
		}

		targetFile := filepath.Join(target, rel)
		exists, err := util.FileExists(targetFile)
		if err != nil {
			return errors.Wrapf(err, "failed to check if file exists %s", targetFile)
		}

		if !exists {
			log.Logger().Warnf("target file %s does not exist so ignoring source", path)
			return nil
		}

		srcNode, err := yaml.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", path)
		}

		targetNode, err := yaml.ReadFile(targetFile)
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", targetFile)
		}

		overlayNode, err := o.createOverlay(srcNode, targetNode, path)
		if err != nil {
			return errors.Wrapf(err, "failed to create a delta node for %s", path)
		}
		if overlayNode == nil {
			log.Logger().Warnf("target file identical for %s so no need for an overlay", path)
			return nil
		}

		overlayFile := filepath.Join(o.OutputDir, rel)
		overlayDir := filepath.Dir(overlayFile)
		err = os.MkdirAll(overlayDir, util.DefaultWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to create output dir %s", overlayDir)
		}

		err = yaml.WriteFile(overlayNode, overlayFile)
		if err != nil {
			return errors.Wrapf(err, "failed to save overlay to %s", overlayFile)
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to generate kustomize overlays to dir %s", dir)
	}
	log.Logger().Infof("created kustomize overlay files at %s", util.ColorInfo(o.OutputDir))
	return nil
}

func (o *Options) createOverlay(srcNode *yaml.RNode, targetNode *yaml.RNode, path string) (*yaml.RNode, error) {
	src := srcNode.YNode()
	target := targetNode.YNode()

	overlay, err := o.removeEqualLeaves(src, target, path, "")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to add overlays to path %s", path)
	}
	if overlay != nil {
		count := 0
		// lets verify we don't only contain mandatory fields
		err = walkMappingNodes(overlay, "", func(node *yaml.Node, jsonPath string) error {
			if jsonPath != "" && jsonPath != "metadata" && util.StringArrayIndex(mandatoryFields, jsonPath) < 0 {
				if count == 0 {
					fmt.Printf("path %s has non mandatory path %s\n", path, jsonPath)
				}
				count++
			}
			return nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed walking mapping nodes %s", path)
		}
		if count == 0 {
			// only mandatory fields so lets assume its empty
			return nil, nil
		}
	}
	if overlay == nil || len(overlay.Content) == 0 {
		return nil, nil
	}
	return targetNode, nil
}

func walkMappingNodes(node *yaml.Node, jsonPath string, fn func(node *yaml.Node, jsonPath string) error) error {
	err := fn(node, jsonPath)
	if err != nil {
		return errors.Wrapf(err, "failed to invoke callback on %s", jsonPath)
	}
	if node.Kind == yaml.MappingNode {
		srcContent := node.Content
		for i := 0; i < len(srcContent)-1; i += 2 {
			sKey := srcContent[i]
			sValue := srcContent[i+1]
			childPath := sKey.Value
			if jsonPath != "" {
				childPath = jsonPath + "." + childPath
			}
			err = walkMappingNodes(sValue, childPath, fn)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *Options) removeEqualLeaves(src *yaml.Node, target *yaml.Node, path string, jsonPath string) (*yaml.Node, error) {
	srcContent := src.Content
	targetContent := target.Content
	if src.Kind != target.Kind {
		return nil, nil
	}
	var replaceTargetIdx []int

	switch src.Kind {
	case yaml.ScalarNode:
		if src.Value == target.Value {
			return nil, nil
		}
		return target, nil

	case yaml.MappingNode:
		for i := 0; i < len(srcContent)-1; i += 2 {
			sKey := srcContent[i]
			sValue := srcContent[i+1]

			j := findMapEntry(sKey, targetContent)
			if j < 0 {
				// TODO should we mark this item as being removed by adding an empty entry?
				continue
			}

			tValue := targetContent[j+1]

			childPath := sKey.Value
			if jsonPath != "" {
				childPath = jsonPath + "." + childPath
			}
			if util.StringArrayIndex(mandatoryFields, childPath) >= 0 {
				continue
			}
			newTValue, err := o.removeEqualLeaves(sValue, tValue, path, childPath)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to process node %s", childPath)
			}
			if newTValue == nil {
				// lets remove this index
				replaceTargetIdx = append(replaceTargetIdx, j)
			}
		}

		// sort the indices in largest first
		sort.Slice(replaceTargetIdx, func(i, j int) bool {
			n1 := replaceTargetIdx[i]
			n2 := replaceTargetIdx[j]
			return n1 > n2
		})

		// lets process the largest index first to avoid index values becoming invalid
		for _, idx := range replaceTargetIdx {
			if idx+2 >= len(targetContent) {
				targetContent = targetContent[0:idx]
			} else {
				targetContent = append(targetContent[0:idx], targetContent[idx+2:]...)
			}
		}

	case yaml.SequenceNode:
		// lets remove this item all the contents are the same
		eq := true
		for i, s := range srcContent {
			if len(targetContent) <= i {
				eq = false
				break
			}
			t := targetContent[i]
			if !scalarsEqual(s, t) {
				childPath := strconv.Itoa(i)
				if jsonPath != "" {
					childPath = jsonPath + "." + childPath
				}
				newTValue, err := o.removeEqualLeaves(s, t, path, childPath)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to process node %s", childPath)
				}
				if newTValue == nil {
					// lets remove this index
					replaceTargetIdx = append(replaceTargetIdx, i)
				} else {
					eq = false
					break
				}
			}
		}
		if eq {
			if len(srcContent) == 0 && len(targetContent) == 0 {
				if src.Value == target.Value {
					return nil, nil
				}
			} else {
				return nil, nil
			}
		}
		// lets iterate in reverse order to preserve the indexes
		for i := len(replaceTargetIdx) - 1; i >= 0; i-- {
			idx := replaceTargetIdx[i]
			targetContent = append(targetContent[0:idx], targetContent[idx+1:]...)
		}
	}

	if len(targetContent) == 0 {
		return nil, nil
	}
	target.Content = targetContent
	return target, nil
}

func findMapEntry(key *yaml.Node, content []*yaml.Node) int {
	for i := 0; i < len(content)-1; i += 2 {
		tKey := content[i]
		if scalarsEqual(key, tKey) {
			return i
		}
	}
	return -1
}

func scalarsEqual(n1 *yaml.Node, n2 *yaml.Node) bool {
	return n1.Kind == yaml.ScalarNode && n2.Kind == yaml.ScalarNode && n1.Value == n2.Value
}
