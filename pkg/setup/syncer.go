package setup

import (
	"context"
	"fmt"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
)

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
