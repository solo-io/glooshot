package checker

import (
	"context"
	"fmt"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/go-utils/contextutils"
)

type failureChecker struct {
	checker ExperimentChecker
}

func NewFailureChecker(checker ExperimentChecker) v1.ApiSyncer {
	return &failureChecker{checker: checker}
}

func (c *failureChecker) Sync(ctx context.Context, snap *v1.ApiSnapshot) error {
	ctx = contextutils.WithLogger(ctx, fmt.Sprintf("failure-checker-sync-%v", snap.Hash()))
	for _, exp := range snap.Experiments {
		exp := exp
		go func() {
			err := c.checker.MonitorExperiment(ctx, exp)
			if err != nil {
				contextutils.LoggerFrom(ctx).Errorf("monitoring experiment %v failed", exp.Metadata.Ref())
			}
		}()
	}
	return nil
}
