package install

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"

	"github.com/solo-io/glooshot/pkg/cli"
	"github.com/spf13/cobra"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/gloo/projects/gloo/cli/pkg/helpers"
)

var _ = Describe("Glooshot CLI", func() {

	var noResourcesTable = `+------------+-----------+--------+
| EXPERIMENT | NAMESPACE | STATUS |
+------------+-----------+--------+
+------------+-----------+--------+`

	BeforeEach(func() {
		helpers.UseMemoryClients()
		_, _ = glooshot("delete experiment --every-resource")
	})

	Context("basic args and flags", func() {
		It("should return help messages without error", func() {
			_, err := glooshot("-h")
			Expect(err).NotTo(HaveOccurred())
			_, err = glooshot("help")
			Expect(err).NotTo(HaveOccurred())
			_, err = glooshot("--help")
			Expect(err).NotTo(HaveOccurred())
		})

		FIt("should return human-friendly errors on bad input", func() {
			_, err := glooshot("--h")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should perform basic create, get, and delete commands", func() {
			out, err := glooshot("get experiments --all-namespaces")
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal(noResourcesTable))

			out, err = glooshot("create experiment -f ../../examples/gs_delay.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal(""))

			out, err = glooshot("get experiments --all-namespaces")
			Expect(err).NotTo(HaveOccurred())
			Expect(out).NotTo(Equal(noResourcesTable))

			out, err = glooshot("delete experiments -n default --all")
			Expect(err).NotTo(HaveOccurred())
			Expect(out).NotTo(Equal(noResourcesTable))

			out, err = glooshot("get experiments --all-namespaces")
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal(noResourcesTable))
		})
	})
})

func glooshot(args string) (string, error) {
	app := cli.App(context.Background(), "testglooshotcli")
	return ExecuteCliOut(app, args)
}

////////////////////////////////////////////////////////////////////////////////
// TODO(mitchdraft) replace with https://github.com/solo-io/go-utils/pull/125 on merge
////////////////////////////////////////////////////////////////////////////////
func ExecuteCli(command *cobra.Command, args string) error {
	command.SetArgs(strings.Split(args, " "))
	return command.Execute()
}

func ExecuteCliOut(command *cobra.Command, args string) (string, error) {
	stdOut := os.Stdout
	stdErr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w
	os.Stderr = w

	command.SetArgs(strings.Split(args, " "))
	err = command.Execute()

	outC := make(chan string)

	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	// back to normal state
	w.Close()
	os.Stdout = stdOut // restoring the real stdout
	os.Stderr = stdErr
	out := <-outC

	return strings.TrimSuffix(out, "\n"), err
}
