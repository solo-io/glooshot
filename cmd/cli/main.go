package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/solo-io/solo-kit/pkg/api/v1/clients"

	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/version"
	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/go-utils/kubeutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/factory"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/kube"
)

func main() {

	if err := Run(); err != nil {
		log.Fatal(err)
	}
}

func Run() error {
	var mode string
	var namespace string
	var name string
	flag.StringVar(&mode, "mode", "", "specify a mode (temporary helper cmd, will be removed)")
	flag.StringVar(&namespace, "namespace", "default", "namespace of experiment")
	flag.StringVar(&name, "name", "", "name of experiment")
	flag.Parse()

	if name == "" {
		log.Fatalf("Must provide an experiment namme")
	}

	cfg, err := kubeutils.GetConfig("", "")
	if err != nil {
		return err
	}
	ctx := contextutils.WithLogger(context.Background(), version.AppName)
	cache := kube.NewKubeCache(ctx)
	rcFactory := &factory.KubeResourceClientFactory{
		Crd:         v1.ExperimentCrd,
		Cfg:         cfg,
		SharedCache: cache,
	}
	client, err := v1.NewExperimentClient(rcFactory)
	client.Register()

	minute := time.Minute
	exp := &v1.Experiment{
		Metadata: core.Metadata{
			Namespace: namespace,
			Name:      name,
		},
	}
	switch mode {
	case "a":
		exp.Spec = &v1.ExperimentSpec{
			StopCondition: &v1.StopCondition{
				Duration: &minute,
				Metric: []*v1.MetricThreshold{
					{
						MetricName: "frank",
						Value:      9000,
					}},
			},
		}
	case "b":
		exp.Spec = &v1.ExperimentSpec{
			StopCondition: &v1.StopCondition{
				Duration: &minute,
				Metric: []*v1.MetricThreshold{
					{
						MetricName: "cores",
						Value:      10,
					}},
			},
		}
	default:
	}
	fmt.Println("attempting to write")
	_, err = client.Write(exp, clients.WriteOpts{})
	return err
}
