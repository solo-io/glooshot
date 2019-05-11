package starter_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	. "github.com/solo-io/glooshot/pkg/starter"
	"github.com/solo-io/glooshot/test/inputs"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/factory"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/memory"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources"
)

var _ = Describe("ExperimentStarter", func() {
	It("starts experiments", func() {

		experimentClientFactory := &factory.MemoryResourceClientFactory{
			Cache: memory.NewInMemoryResourceCache(),
		}
		experimentClient, err := v1.NewExperimentClient(experimentClientFactory)
		Expect(err).NotTo(HaveOccurred())

		exp1 := inputs.MakeExperiment("h")
		exp1.Result.TimeStarted = nil
		exp1.Result.State = v1.ExperimentResult_Pending
		exp1, err = experimentClient.Write(exp1, clients.WriteOpts{})
		Expect(err).NotTo(HaveOccurred())

		exp2 := resources.Clone(exp1).(*v1.Experiment)
		exp2.Metadata.Name = "somethingelse"
		exp2, err = experimentClient.Write(exp2, clients.WriteOpts{})
		Expect(err).NotTo(HaveOccurred())

		starter := NewExperimentStarter(experimentClient)

		err = starter.Sync(context.TODO(), &v1.ApiSnapshot{Experiments: v1.ExperimentList{exp1, exp2}})
		Expect(err).NotTo(HaveOccurred())

		exp1, err = experimentClient.Read(exp1.Metadata.Namespace, exp1.Metadata.Name, clients.ReadOpts{})
		Expect(err).NotTo(HaveOccurred())

		Expect(exp1.Result.State).To(Equal(v1.ExperimentResult_Started))
		Expect(exp1.Result.TimeStarted).NotTo(BeNil())

		exp2, err = experimentClient.Read(exp2.Metadata.Namespace, exp2.Metadata.Name, clients.ReadOpts{})
		Expect(err).NotTo(HaveOccurred())

		Expect(exp2.Result.State).To(Equal(v1.ExperimentResult_Started))
		Expect(exp2.Result.TimeStarted).NotTo(BeNil())
	})
})
