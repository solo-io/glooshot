package translator

import (
	"context"
	"fmt"

	"github.com/solo-io/glooshot/pkg/setup/options"

	"github.com/gogo/protobuf/proto"

	"github.com/pkg/errors"

	sgv1 "github.com/solo-io/supergloo/pkg/api/v1"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
)

const RoutingRuleLabelKey = "glooshot-experiment"

func applyCreatedByLabels(labels map[string]string) {
	labels["created_by"] = "glooshot"
}

type glooshotSyncer struct {
	expClient    v1.ExperimentClient
	rrClient     sgv1.RoutingRuleClient
	rrReconciler sgv1.RoutingRuleReconciler
	meshClient   sgv1.MeshClient
	opts         options.Opts
}

func (g *glooshotSyncer) Sync(ctx context.Context, snap *v1.ApiSnapshot) error {
	desired, err := g.translateExperimentsToRoutingRules(snap.Experiments)
	if err != nil {
		return err
	}
	labels := map[string]string{}
	applyCreatedByLabels(labels)
	if err := g.rrReconciler.Reconcile("", desired, nil, clients.ListOpts{Ctx: ctx, Selector: labels}); err != nil {
		return err
	}
	return nil
}

func NewSyncer(expClient v1.ExperimentClient, rrClient sgv1.RoutingRuleClient, meshClient sgv1.MeshClient, opts options.Opts) *glooshotSyncer {
	return &glooshotSyncer{
		expClient:    expClient,
		rrClient:     rrClient,
		rrReconciler: sgv1.NewRoutingRuleReconciler(rrClient),
		meshClient:   meshClient,
		opts:         opts,
	}
}

func (g *glooshotSyncer) translateExperimentsToRoutingRules(exps v1.ExperimentList) (sgv1.RoutingRuleList, error) {
	rrs := sgv1.RoutingRuleList{}
	for _, exp := range exps {
		if exp.Spec == nil || len(exp.Spec.Faults) == 0 {
			continue
		}
		for i := range exp.Spec.Faults {
			rr, err := g.translateToRoutingRule(exp, i)
			if err != nil {
				return nil, err
			}
			rrs = append(rrs, rr)
		}
	}
	return rrs, nil
}

func (g *glooshotSyncer) translateToRoutingRule(exp *v1.Experiment, index int) (*sgv1.RoutingRule, error) {
	expName := exp.Metadata.Name
	namespace := exp.Metadata.Namespace
	wrap := func(e error) error {
		return errors.Wrapf(e, "for experiment: %v.%v, fault index: %v", namespace, expName, index)
	}
	rrName := fmt.Sprintf("%v-%v", expName, index)
	labels := LabelsForRoutingRule(expName)
	f := exp.Spec.Faults[index]
	targetMesh, err := g.getTargetMesh(exp.Spec.TargetMesh)
	if err != nil {
		return nil, wrap(err)
	}
	var ss *sgv1.PodSelector
	if f.OriginServices != nil {
		ss, err = selectorFromResourceRef(f.OriginServices)
		if err != nil {
			return nil, wrap(err)
		}
	}
	var ds *sgv1.PodSelector
	if f.DestinationServices != nil {
		ds, err = selectorFromResourceRef(f.DestinationServices)
		if err != nil {
			return nil, wrap(err)
		}
	}
	spec, err := translateFaultToSpec(f.Fault)
	if err != nil {
		return nil, wrap(err)
	}
	return &sgv1.RoutingRule{
		Metadata: core.Metadata{
			Name: rrName,
			// store the faults in the same ns as the experiment
			Namespace: namespace,
			Labels:    labels,
		},
		TargetMesh:          targetMesh,
		SourceSelector:      ss,
		DestinationSelector: ds,
		Spec:                spec,
	}, nil
}

func (g *glooshotSyncer) getTargetMesh(entry *core.ResourceRef) (*core.ResourceRef, error) {
	// user provided a mesh spec on the Experiment, verify that it exists
	if entry != nil {
		if _, err := g.meshClient.Read(entry.Namespace, entry.Name, clients.ReadOpts{}); err != nil {
			return nil, err
		}
		return entry, nil
	}

	// user did not provide a mesh spec, try to choose a default
	meshes, err := g.meshClient.List(g.opts.MeshResourceNamespace, clients.ListOpts{})
	if err != nil {
		return nil, err
	}
	if len(meshes) == 0 {
		return nil, fmt.Errorf("no mesh target specified and "+
			"no meshes found in namespace: %v",
			g.opts.MeshResourceNamespace)
	}
	if len(meshes) > 1 {
		return nil, fmt.Errorf("no target mesh specified and "+
			"cannot choose default among the multiple (%v) meshes found in namespace: %v",
			len(meshes),
			g.opts.MeshResourceNamespace)
	}

	meshRef := meshes[0].Metadata.Ref()
	return &meshRef, nil
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

func LabelsForRoutingRule(expName string) map[string]string {
	labels := map[string]string{RoutingRuleLabelKey: expName}
	applyCreatedByLabels(labels)
	return labels
}
