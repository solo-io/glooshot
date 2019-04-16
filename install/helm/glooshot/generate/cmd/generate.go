package main

import (
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/solo-io/glooshot/install/helm/glooshot/generate"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/solo-io/solo-kit/pkg/utils/log"
)

var (
	valuesTemplate = "install/helm/glooshot/values-template.yaml"
	valuesOutput   = "install/helm/glooshot/values.yaml"
	chartTemplate  = "install/helm/glooshot/Chart-template.yaml"
	chartOutput    = "install/helm/glooshot/Chart.yaml"

	ifNotPresent = "IfNotPresent"
)

func main() {
	var version, repoPrefixOverride = "", ""
	if len(os.Args) < 2 {
		panic("Must provide version as argument")
	} else {
		version = os.Args[1]

		if len(os.Args) == 3 {
			repoPrefixOverride = os.Args[2]
		}

	}
	log.Printf("Generating helm files.")
	if err := generateValuesYaml(version, repoPrefixOverride); err != nil {
		log.Fatalf("generating values.yaml failed!: %v", err)
	}
	if err := generateChartYaml(version); err != nil {
		log.Fatalf("generating Chart.yaml failed!: %v", err)
	}
}

func readYaml(path string, obj interface{}) error {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.Wrapf(err, "failed reading server config file: %s", path)
	}

	if err := yaml.Unmarshal(bytes, obj); err != nil {
		return errors.Wrap(err, "failed parsing configuration file")
	}

	return nil
}

func writeYaml(obj interface{}, path string) error {
	bytes, err := yaml.Marshal(obj)
	if err != nil {
		return errors.Wrapf(err, "failed marshaling config struct")
	}

	err = ioutil.WriteFile(path, bytes, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "failing writing config file")
	}
	return nil
}

func readValuesTemplate() (*generate.Config, error) {
	var config generate.Config
	if err := readYaml(valuesTemplate, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func generateValuesYaml(version, repositoryPrefix string) error {
	cfg, err := readValuesTemplate()
	if err != nil {
		return err
	}

	cfg.Glooshot.Deployment.Image.Tag = version

	if strings.HasSuffix(version, "dev") {
		cfg.Glooshot.Deployment.Image.PullPolicy = ifNotPresent
	}

	if repositoryPrefix != "" {
		cfg.Glooshot.Deployment.Image.Repository = replacePrefix(cfg.Glooshot.Deployment.Image.Repository, repositoryPrefix)
	}

	return writeYaml(cfg, valuesOutput)
}

func generateChartYaml(version string) error {
	var chart generate.Chart
	if err := readYaml(chartTemplate, &chart); err != nil {
		return err
	}

	chart.Version = version

	return writeYaml(&chart, chartOutput)
}

// We want to turn "quay.io/solo-io/foo" into "<newPrefix>/foo".
func replacePrefix(repository, newPrefix string) string {
	// Remove trailing slash, if present
	newPrefix = strings.TrimSuffix(newPrefix, "/")
	return strings.Join([]string{newPrefix, path.Base(repository)}, "/")
}
