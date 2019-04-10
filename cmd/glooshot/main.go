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
	fmt.Println("todo - apiserver runner")
	http.HandleFunc("/todo", handleTODO)
	dh := defaultHandler{}
	http.Handle("/", dh)
	go http.ListenAndServe("localhost:8085", nil)
	log.Fatal(Run())

}

func handleTODO(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "TODO")
}

type defaultHandler struct{}

func (d defaultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello from default")
}

func Run() error {
	start := time.Now()
	checkpoint.CallCheck(version.AppName, version.Version, start)
	// prevent panic if multiple flag.Parse called concurrently
	flag.Parse()
	cfg, err := kubeutils.GetConfig("", "")
	if err != nil {
		return err
	}
	ctx := contextutils.WithLogger(context.Background(), version.AppName)
	cache := kube.NewKubeCache(ctx)
	rcFactory := &factory.KubeResourceClientFactory{
		Crd:             v1.ExperimentCrd,
		Cfg:             cfg,
		SharedCache:     cache,
		SkipCrdCreation: true,
	}
	client, err := v1.NewExperimentClient(rcFactory)
	client.Register()

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
