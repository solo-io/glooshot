package setup

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/version"
	"github.com/solo-io/go-checkpoint"
	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/go-utils/kubeutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/factory"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/kube"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources"
)

type StatsHandler struct {
	ctx context.Context
}

func NewStatsHandler() StatsHandler {
	return StatsHandler{
		ctx: contextutils.WithLogger(context.Background(), version.AppName),
	}
}

func (d StatsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Glooshot stats\n")
	ok := func(err error) bool {
		if err != nil {
			contextutils.LoggerFrom(d.ctx).Errorf("error getting client: %v", err)
			fmt.Fprintf(w, "error in stats handler %v", err)
			return false
		}
		return true
	}
	client, err := GetExperimentClient(d.ctx, true)
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
	fmt.Fprintf(w, "%v", summary)
}

func getExperimentNamespaces(ctx context.Context) []string {
	contextutils.LoggerFrom(ctx).Errorf("TODO: implement getExperimentNamespaces")
	return []string{"default"}
}

func Run() error {
	start := time.Now()
	checkpoint.CallCheck(version.AppName, version.Version, start)
	flag.Parse()

	sh := NewStatsHandler()
	http.Handle("/", sh)
	go http.ListenAndServe("localhost:8085", nil)

	ctx := contextutils.WithLogger(context.Background(), version.AppName)
	client, err := GetExperimentClient(ctx, true)
	if err != nil {
		return err
	}
	syncer := NewSyncer(client)
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
	expClient v1.ExperimentClient
	lastHash  uint64
	last      map[string]string
}

func (g glooshotSyncer) Sync(ctx context.Context, snap *v1.ApiSnapshot) error {
	hash := snap.Hash()
	if hash == g.lastHash {
		return nil
	}
	g.lastHash = hash
	unchangedCount := 0
	updatedCount := 0
	createdCount := 0
	deletedCount := 0
	existingKeys := make(map[string]bool)
	for _, exp := range snap.Experiments.List() {
		key := resources.Key(exp)
		val, ok := g.last[key]
		if ok {
			if val == exp.Metadata.ResourceVersion {
				unchangedCount++
			} else {
				updatedCount++
				fmt.Printf("Updated experiment: %v\n", key)
			}
		} else {
			createdCount++
			fmt.Printf("Created experiment: %v\n", key)
			if err := g.mutateNewlyCreatedExperiments(exp); err != nil {
				contextutils.LoggerFrom(ctx).Errorf("sync mutation failed on %v: %v", key, err)
			}
		}
		existingKeys[key] = true
		g.last[key] = exp.Metadata.ResourceVersion
	}
	for k := range g.last {
		if _, ok := existingKeys[k]; !ok {
			delete(g.last, k)
			deletedCount++
			fmt.Printf("Deleted experiment: %v\n", k)
		}
	}
	fmt.Printf("Experiments: Created: %v, updated: %v, deleted %v, unchanged: %v\n",
		createdCount,
		updatedCount,
		deletedCount,
		unchangedCount)
	return nil
}

func (g glooshotSyncer) getClient() v1.ExperimentClient {
	if g.expClient != nil {
		return g.expClient
	}
	return g.expClient
}

func (g glooshotSyncer) mutateNewlyCreatedExperiments(exp *v1.Experiment) error {
	exp.Status.State = core.Status_Accepted
	_, err := g.expClient.Write(exp, clients.WriteOpts{OverwriteExisting: true})
	return err
}

func NewSyncer(client v1.ExperimentClient) glooshotSyncer {
	return glooshotSyncer{
		expClient: client,
		lastHash:  0,
		last:      make(map[string]string),
	}
}

func GetExperimentClient(ctx context.Context, skipCrdCreation bool) (v1.ExperimentClient, error) {
	cfg, err := kubeutils.GetConfig("", "")
	if err != nil {
		return nil, err
	}
	cache := kube.NewKubeCache(ctx)
	rcFactory := &factory.KubeResourceClientFactory{
		Crd:             v1.ExperimentCrd,
		Cfg:             cfg,
		SharedCache:     cache,
		SkipCrdCreation: skipCrdCreation,
	}
	client, err := v1.NewExperimentClient(rcFactory)
	client.Register()
	return client, nil
}
