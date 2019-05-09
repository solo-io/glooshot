package e2e

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/solo-io/glooshot/pkg/cli/gsutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/setup"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
)

var _ = Describe("Glooshot", func() {

	var (
		ctx       context.Context
		client    v1.ExperimentClient
		namespace = "default"
		name      = "testexperiment"
		url       = "http://localhost:8085"
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		client, err = gsutil.GetExperimentClient(ctx, false)
		Expect(err).NotTo(HaveOccurred())
		go setup.Run(ctx)
	})

	AfterEach(func() {
		// run delete in case the test exited early
		client.Delete(namespace, name, clients.DeleteOpts{})
	})

	It("should watch for experiment crds", func() {
		exp := getNewExperiment(namespace, name)
		_, err := client.Write(exp, clients.WriteOpts{})
		Expect(err).NotTo(HaveOccurred())

		time.Sleep(time.Second)
		body, err := curl(url)
		Expect(err).NotTo(HaveOccurred())
		str := `Glooshot stats
Count: 1
Experiment Summary
default, testexperiment: Accepted
`
		ExpectWithOffset(1, body).To(Equal(str))

		err = client.Delete(namespace, name, clients.DeleteOpts{})
		Expect(err).NotTo(HaveOccurred())
	})
})

func getNewExperiment(namespace, name string) *v1.Experiment {
	return &v1.Experiment{
		Metadata: core.Metadata{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// TODO(mitchdraft) migrate this to go-utils https://github.com/solo-io/glooshot/issues/16
func curl(url string) (string, error) {
	body := bytes.NewReader([]byte(url))
	req, err := http.NewRequest("GET", url, body)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	p := new(bytes.Buffer)
	_, err = io.Copy(p, resp.Body)
	defer resp.Body.Close()

	return p.String(), nil
}
