package setup

import (
	"context"
	"fmt"

	sgv1 "github.com/solo-io/supergloo/pkg/api/v1"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
)

type glooshotSyncer struct {
	expClient    v1.ExperimentClient
	rrClient     sgv1.RoutingRuleClient
	rrReconciler sgv1.RoutingRuleReconciler
	last         map[string]string
}

func (g glooshotSyncer) Sync(ctx context.Context, snap *v1.ApiSnapshot) error {
	// Will need to update this with the solo-kit update
	expsByNamespace := snap.Experiments
	desired := sgv1.RoutingRuleList{}
	for ns, exps := range expsByNamespace {
		desired = append(desired, translateExperimentsToRoutingRules(exps)...)
		if err := g.rrReconciler.Reconcile(ns, desired, nil, clients.ListOpts{}); err != nil {
			return err
		}
	}
	return nil
}

func NewSyncer(expClient v1.ExperimentClient, rrClient sgv1.RoutingRuleClient) glooshotSyncer {
	return glooshotSyncer{
		expClient:    expClient,
		rrClient:     rrClient,
		rrReconciler: sgv1.NewRoutingRuleReconciler(rrClient),
		last:         make(map[string]string),
	}
}

func translateExperimentsToRoutingRules(exps v1.ExperimentList) sgv1.RoutingRuleList {
	rrs := sgv1.RoutingRuleList{}
	for _, exp := range exps {
		for i := range exp.Spec.Faults {
			rr := translateToRoutingRule(exp, i)
			rrs = append(rrs, rr)
		}
	}
	return rrs
}

func translateToRoutingRule(exp *v1.Experiment, index int) *sgv1.RoutingRule {
	expName := exp.Metadata.Name
	namespace := exp.Metadata.Namespace
	rrName := fmt.Sprintf("%v-%v", expName, index)
	labels := labelsForRoutingRule(expName)
	f := exp.Spec.Faults[index]
	return &sgv1.RoutingRule{
		Metadata: core.Metadata{
			Name: rrName,
			// store the faults in the same ns as the experiment
			Namespace: namespace,
			Labels:    labels,
		},
		TargetMesh:          exp.Result.TargetMesh,
		SourceSelector:      selectorFromResourceRef(f.OriginServices),
		DestinationSelector: selectorFromResourceRef(f.DestinationServices),
		Spec:                translateFaultToSpec(f.Fault),
	}
}

func selectorFromResourceRef(refs []*core.ResourceRef) *sgv1.PodSelector {
	upstreams := []core.ResourceRef{}
	for _, point := range refs {
		upstreams = append(upstreams, *point)
	}
	return &sgv1.PodSelector{
		SelectorType: &sgv1.PodSelector_UpstreamSelector_{
			UpstreamSelector: &sgv1.PodSelector_UpstreamSelector{
				Upstreams: upstreams,
			},
		},
	}
}

func translateFaultToSpec(fault *sgv1.FaultInjection) *sgv1.RoutingRuleSpec {
	return &sgv1.RoutingRuleSpec{
		RuleType: &sgv1.RoutingRuleSpec_FaultInjection{
			FaultInjection: fault,
		},
	}
}

func labelsForRoutingRule(expName string) map[string]string {
	return map[string]string{"experiment": expName}
}
