package translator

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"

	"github.com/pkg/errors"

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
	desired, err := translateExperimentsToRoutingRules(snap.Experiments)
	if err != nil {
		return err
	}
	if err := g.rrReconciler.Reconcile("", desired, nil, clients.ListOpts{}); err != nil {
		return err
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

func translateExperimentsToRoutingRules(exps v1.ExperimentList) (sgv1.RoutingRuleList, error) {
	rrs := sgv1.RoutingRuleList{}
	for _, exp := range exps {
		if exp.Spec == nil || len(exp.Spec.Faults) == 0 {
			continue
		}
		for i := range exp.Spec.Faults {
			rr, err := translateToRoutingRule(exp, i)
			if err != nil {
				return nil, err
			}
			rrs = append(rrs, rr)
		}
	}
	return rrs, nil
}

func translateToRoutingRule(exp *v1.Experiment, index int) (*sgv1.RoutingRule, error) {
	expName := exp.Metadata.Name
	namespace := exp.Metadata.Namespace
	rrName := fmt.Sprintf("%v-%v", expName, index)
	labels := labelsForRoutingRule(expName)
	f := exp.Spec.Faults[index]
	spec, err := translateFaultToSpec(f.Fault)
	wrap := func(e error) error {
		return errors.Wrapf(e, "for experiment: %v.%v, fault index: %v", namespace, expName, index)
	}
	if err != nil {
		return nil, wrap(err)
	}
	ss := &sgv1.PodSelector{}
	if f.OriginServices != nil {
		ss, err = selectorFromResourceRef(f.OriginServices)
		if err != nil {
			return nil, wrap(err)
		}
	}
	ds := &sgv1.PodSelector{}
	if f.DestinationServices != nil {
		ds, err = selectorFromResourceRef(f.DestinationServices)
		if err != nil {
			return nil, wrap(err)
		}
	}
	return &sgv1.RoutingRule{
		Metadata: core.Metadata{
			Name: rrName,
			// store the faults in the same ns as the experiment
			Namespace: namespace,
			Labels:    labels,
		},
		TargetMesh:          exp.Spec.TargetMesh,
		SourceSelector:      ss,
		DestinationSelector: ds,
		Spec:                spec,
	}, nil
}

func selectorFromResourceRef(refs []*core.ResourceRef) (*sgv1.PodSelector, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	upstreams := []core.ResourceRef{}
	for _, point := range refs {
		if point == nil {
			return nil, fmt.Errorf("nil resource ref cannot be translated to a pod selector")
		}
		upstreams = append(upstreams, *point)
	}
	return &sgv1.PodSelector{
		SelectorType: &sgv1.PodSelector_UpstreamSelector_{
			UpstreamSelector: &sgv1.PodSelector_UpstreamSelector{
				Upstreams: upstreams,
			},
		},
	}, nil
}

func translateFaultToSpec(fault *sgv1.FaultInjection) (*sgv1.RoutingRuleSpec, error) {
	if fault == nil || proto.Equal(fault, &sgv1.FaultInjection{}) {
		return nil, fmt.Errorf("empty fault injection detected")
	}
	return &sgv1.RoutingRuleSpec{
		RuleType: &sgv1.RoutingRuleSpec_FaultInjection{
			FaultInjection: fault,
		},
	}, nil
}

func labelsForRoutingRule(expName string) map[string]string {
	return map[string]string{"experiment": expName}
}
