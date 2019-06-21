package translator

import (
	"context"
	"time"

	"github.com/solo-io/solo-kit/pkg/api/v1/clients"

	"github.com/solo-io/glooshot/pkg/api/v1/mocks"
	"github.com/solo-io/glooshot/pkg/setup/options"
	sgmock "github.com/solo-io/supergloo/pkg/api/v1/mocks"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/go-utils/testutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
	sgv1 "github.com/solo-io/supergloo/pkg/api/v1"
)

var _ = Describe("translator", func() {
	var (
		nilFault        *sgv1.FaultInjection
		emptyFault      *sgv1.FaultInjection
		basicAbortFault *sgv1.FaultInjection

		destination1          *core.ResourceRef
		destination2          *core.ResourceRef
		basicFailureCondition *v1.FailureCondition
		duration1             time.Duration
		basicExperiment       *v1.Experiment
		syncer                glooshotSyncer
		mockCtrl              *gomock.Controller
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		syncer = newMockSyncer(mockCtrl)
		nilFault = &sgv1.FaultInjection{}
		nilFault = nil

		emptyFault = &sgv1.FaultInjection{}

		basicAbortFault = &sgv1.FaultInjection{
			FaultInjectionType: &sgv1.FaultInjection_Abort_{
				Abort: &sgv1.FaultInjection_Abort{
					ErrorType: &sgv1.FaultInjection_Abort_HttpStatus{
						HttpStatus: 404,
					},
				},
			},
			Percentage: 50.0,
		}

		destination1 = &core.ResourceRef{"name1", "default"}
		destination2 = &core.ResourceRef{"name2", "default"}
		basicFailureCondition = &v1.FailureCondition{
			Trigger: &v1.FailureCondition_Trigger{
				FailureTrigger: &v1.FailureCondition_Trigger_Prometheus{
					Prometheus: &v1.PrometheusTrigger{
						QueryType:          nil,
						ThresholdValue:     0,
						ComparisonOperator: "",
					},
				},
			},
		}
		duration1 = time.Hour
		basicExperiment = &v1.Experiment{
			Metadata: core.Metadata{
				Name:      "basic",
				Namespace: "default",
			},
			Spec: &v1.ExperimentSpec{
				Faults: []*v1.ExperimentSpec_InjectedFault{{
					OriginServices:      []*core.ResourceRef{destination1},
					DestinationServices: []*core.ResourceRef{destination2},
					Fault:               basicAbortFault,
				}},
				FailureConditions: []*v1.FailureCondition{basicFailureCondition},
				Duration:          &duration1,
				TargetMesh: &core.ResourceRef{
					Name:      "basicmesh",
					Namespace: "default",
				},
			},
			Result: v1.ExperimentResult{
				State:         0,
				FailureReport: map[string]string{},
			},
		}

	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("should convert a single fault", func() {
		translated, err := translateFaultToSpec(nilFault)
		Expect(err).To(HaveOccurred())

		translated, err = translateFaultToSpec(emptyFault)
		Expect(err).To(HaveOccurred())

		basicRRSpec := &sgv1.RoutingRuleSpec{
			RuleType: &sgv1.RoutingRuleSpec_FaultInjection{
				FaultInjection: basicAbortFault,
			},
		}
		translated, err = translateFaultToSpec(basicAbortFault)
		Expect(err).NotTo(HaveOccurred())
		testutils.ExpectEqualProtoMessages(translated, basicRRSpec)
	})

	It("should get a selector from a resource ref", func() {
		ref := core.ResourceRef{"abc", "default"}
		refs := []*core.ResourceRef{&ref}
		Expect(selectorFromResourceRef(refs)).To(Equal(&sgv1.PodSelector{
			SelectorType: &sgv1.PodSelector_UpstreamSelector_{
				UpstreamSelector: &sgv1.PodSelector_UpstreamSelector{
					Upstreams: []core.ResourceRef{ref},
				},
			},
		}))
	})

	It("should translate routing rule", func() {
		mockMesh := sgmock.NewMockMeshClient(mockCtrl)
		mockMesh.EXPECT().Read("default", "basicmesh", clients.ReadOpts{})
		syncer.meshClient = mockMesh
		rr, err := syncer.translateToRoutingRule(context.Background(), basicExperiment, 0)
		Expect(err).NotTo(HaveOccurred())
		expected := &sgv1.RoutingRule{
			Status: core.Status{
				State:               0,
				Reason:              "",
				ReportedBy:          "",
				SubresourceStatuses: nil,
			},
			Metadata: core.Metadata{
				Name:            "basic-0",
				Namespace:       "default",
				Cluster:         "",
				ResourceVersion: "",
				Labels: map[string]string{
					RoutingRuleLabelKey: "basic",
					"created_by":        "glooshot",
				},
				Annotations: nil,
			},
			TargetMesh: &core.ResourceRef{Name: "basicmesh", Namespace: "default"},
			SourceSelector: &sgv1.PodSelector{
				SelectorType: &sgv1.PodSelector_UpstreamSelector_{
					UpstreamSelector: &sgv1.PodSelector_UpstreamSelector{
						Upstreams: []core.ResourceRef{
							{Name: "name1", Namespace: "default"}},
					},
				},
			},
			DestinationSelector: &sgv1.PodSelector{
				SelectorType: &sgv1.PodSelector_UpstreamSelector_{
					UpstreamSelector: &sgv1.PodSelector_UpstreamSelector{
						Upstreams: []core.ResourceRef{{Name: "name2", Namespace: "default"}},
					},
				},
			},
			Spec: &sgv1.RoutingRuleSpec{
				RuleType: &sgv1.RoutingRuleSpec_FaultInjection{
					FaultInjection: &sgv1.FaultInjection{
						FaultInjectionType: &sgv1.FaultInjection_Abort_{
							Abort: &sgv1.FaultInjection_Abort{
								ErrorType: &sgv1.FaultInjection_Abort_HttpStatus{HttpStatus: 404},
							},
						},
						Percentage: 50,
					},
				},
			},
		}
		testutils.ExpectEqualProtoMessages(rr, expected)

	})

})

// populates clients with mocks
// override as needed with mocks that provide mock.EXPECT().SomeMethod("some","args")
// these placeholders will at least notify you where you need to add overrides when/if they fail
func newMockSyncer(ctrl *gomock.Controller) glooshotSyncer {
	return glooshotSyncer{
		expClient: mocks.NewMockExperimentClient(ctrl),
		rrClient:  sgmock.NewMockRoutingRuleClient(ctrl),
		//rrReconciler: nil, // Mock as needed
		meshClient: sgmock.NewMockMeshClient(ctrl),
		opts:       options.Opts{},
	}
}
