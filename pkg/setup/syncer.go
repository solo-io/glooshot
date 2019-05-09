package setup

import (
	"context"

	"go.opencensus.io/trace"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
)

type glooshotSyncer struct {
	expClient v1.ExperimentClient
	last      map[string]string
}

func (g glooshotSyncer) Sync(ctx context.Context, snap *v1.ApiSnapshot) error {
	unchangedCount := 0
	updatedCount := 0
	createdCount := 0
	deletedCount := 0
	existingKeys := make(map[string]bool)
	for _, exp := range snap.Experiments {
		key := resources.Key(exp)
		val, ok := g.last[key]
		if ok {
			if val == exp.Metadata.ResourceVersion {
				unchangedCount++
			} else {
				updatedCount++
				contextutils.LoggerFrom(ctx).Infow("Experiment", "updated", key)
			}
		} else {
			createdCount++
			contextutils.LoggerFrom(ctx).Infow("Experiment", "created", key)
			go func() {
				if err := g.mutateNewlyCreatedExperiments(ctx, exp); err != nil {
					contextutils.LoggerFrom(ctx).Errorf("sync mutation failed on %v: %v", key, err)
				}
			}()
		}
		existingKeys[key] = true
		g.last[key] = exp.Metadata.ResourceVersion
	}
	for k := range g.last {
		if _, ok := existingKeys[k]; !ok {
			delete(g.last, k)
			deletedCount++
			contextutils.LoggerFrom(ctx).Infow("Experiment", "deleted", k)
		}
	}
	contextutils.LoggerFrom(ctx).Infow("Experiment",
		"created", createdCount,
		"updated", updatedCount,
		"deleted", deletedCount,
		"unchanged", unchangedCount)
	return nil
}

func (g glooshotSyncer) mutateNewlyCreatedExperiments(ctx context.Context, exp *v1.Experiment) error {
	_, span := trace.StartSpan(ctx, "glooshot.solo.io.mutateNewlyCreatedExperiments")
	defer span.End()
	exp.Status.State = core.Status_Accepted
	_, err := g.expClient.Write(exp, clients.WriteOpts{OverwriteExisting: true})
	return err
}

func NewSyncer(client v1.ExperimentClient) glooshotSyncer {
	return glooshotSyncer{
		expClient: client,
		last:      make(map[string]string),
	}
}
