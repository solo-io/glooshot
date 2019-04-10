package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/solo-io/solo-kit/pkg/api/v1/resources"

	"github.com/solo-io/solo-kit/pkg/api/v1/clients"

	"github.com/solo-io/go-checkpoint"

	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/go-utils/kubeutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/kube"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/version"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/factory"
)

func main() {
	sh := NewStatsHandler()
	http.Handle("/", sh)
	go http.ListenAndServe("localhost:8085", nil)
	log.Fatal(Run())

}

type statsHandler struct {
	ctx context.Context
}

func NewStatsHandler() statsHandler {
	return statsHandler{
		ctx: contextutils.WithLogger(context.Background(), version.AppName),
	}
}

func (d statsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Glooshot stats\n")
	ok := func(err error) bool {
		if err != nil {
			contextutils.LoggerFrom(d.ctx).Errorf("error getting client: %v", err)
			fmt.Fprintf(w, "error in stats handler %v", err)
			return false
		}
		return true
	}
	client, err := getExperimentClient(d.ctx)
	if !ok(err) {
		return
	}
	experimentNamespaces := getExperimentNamespaces(d.ctx)
	expCount := 0
	summary := "Experiment Summary\n"
	for _, ns := range experimentNamespaces {
		exps, err := client.List(ns, clients.ListOpts{})
		if !ok(err) {
			return
		}
		for _, exp := range exps {
			summary += fmt.Sprintf("%v, %v: %v\n",
				exp.Metadata.Namespace,
				exp.Metadata.Name,
				exp.Status.State.String())
			expCount++
		}
	}
	fmt.Fprintf(w, "Count: %v\n", expCount)
	fmt.Fprintf(w, "%v\n", summary)
}

func getExperimentNamespaces(ctx context.Context) []string {
	contextutils.LoggerFrom(ctx).Errorf("TODO: implement getExperimentNamespaces")
	return []string{"default"}
}

func Run() error {
	start := time.Now()
	checkpoint.CallCheck(version.AppName, version.Version, start)
	flag.Parse()

	ctx := contextutils.WithLogger(context.Background(), version.AppName)
	client, err := getExperimentClient(ctx)
	if err != nil {
		return err
	}
	syncer := NewSyncer()
	el := v1.NewApiEventLoop(v1.NewApiEmitter(client), syncer)
	errs, err := el.Run([]string{}, clients.WatchOpts{
		Ctx:         ctx,
		RefreshRate: time.Second,
	})

	for err := range errs {
		contextutils.LoggerFrom(ctx).Fatalf("error in setup: %v", err)
	}
	return nil
}

type glooshotSyncer struct {
	lastHash uint64
	last     map[string]string
}

func (g glooshotSyncer) Sync(ctx context.Context, snap *v1.ApiSnapshot) error {
	hash := snap.Hash()
	if hash == g.lastHash {
		return nil
	}
	g.lastHash = hash
	for _, exp := range snap.Experiments.List() {
		key := resources.Key(exp)
		val, ok := g.last[key]
		if ok && val == exp.Metadata.ResourceVersion {
			continue
		}
		g.last[key] = exp.Metadata.ResourceVersion
		if ok {
			fmt.Printf("Updated experiment: %v\n", key)
		} else {
			fmt.Printf("Received new experiment: %v\n", key)
		}

	}
	return nil
}

func NewSyncer() glooshotSyncer {
	return glooshotSyncer{
		lastHash: 0,
		last:     make(map[string]string),
	}
}

func getExperimentClient(ctx context.Context) (v1.ExperimentClient, error) {
	cfg, err := kubeutils.GetConfig("", "")
	if err != nil {
		return nil, err
	}
	cache := kube.NewKubeCache(ctx)
	rcFactory := &factory.KubeResourceClientFactory{
		Crd:             v1.ExperimentCrd,
		Cfg:             cfg,
		SharedCache:     cache,
		SkipCrdCreation: true,
	}
	client, err := v1.NewExperimentClient(rcFactory)
	client.Register()
	return client, nil
}
