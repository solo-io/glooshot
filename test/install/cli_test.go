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
	var standardCobraHelpBlockMatcher = MatchRegexp("Available Commands:")

	BeforeEach(func() {
		helpers.UseMemoryClients()
		_, _, _ = glooshot("delete experiment --every-resource")
	})

	Context("basic args and flags", func() {
		It("should return help messages without error", func() {
			_, _, err := glooshot("-h")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = glooshot("help")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = glooshot("--help")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should perform basic create, get, and delete commands", func() {
			out, stdErr, err := glooshot("get experiments --all-namespaces")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).NotTo(HaveOccurred())
			Expect(out).To(Equal(noResourcesTable))

			out, stdErr, err = glooshot("create experiment -f ../../examples/gs_delay.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).NotTo(HaveOccurred())
			Expect(out).To(Equal(""))

			out, stdErr, err = glooshot("get experiments --all-namespaces")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).NotTo(HaveOccurred())
			Expect(out).NotTo(Equal(noResourcesTable))

			out, stdErr, err = glooshot("delete experiments -n default --all")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).NotTo(HaveOccurred())
			Expect(out).NotTo(Equal(noResourcesTable))

			out, stdErr, err = glooshot("get experiments --all-namespaces")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).NotTo(HaveOccurred())
			Expect(out).To(Equal(noResourcesTable))
		})
	})

	Context("expect human-friendly errors", func() {

		FIt("should return human-friendly errors on bad input", func() {
			stdOut, stdErr, err := glooshot("--h")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unknown flag: --h"))
			Expect(stdErr).NotTo(HaveOccurred())
			Expect(stdOut).To(standardCobraHelpBlockMatcher)
		})

		It("should return human-friendly errors on bad input", func() {
			stdOut, stdErr, err := glooshot("")
			Expect(err).To(HaveOccurred())
			Expect(stdErr).NotTo(HaveOccurred())
			Expect(err.Error()).To(Equal("unknown flag: --h"))
			Expect(stdOut).To(standardCobraHelpBlockMatcher)
		})

	})
})

func glooshot(args string) (string, string, error) {
	app := cli.App(context.Background(), "testglooshotcli")
	return ExecuteCliOutErr(app, args)
}

////////////////////////////////////////////////////////////////////////////////
// TODO(mitchdraft) replace with https://github.com/solo-io/go-utils/pull/125 on merge
////////////////////////////////////////////////////////////////////////////////
func ExecuteCli(command *cobra.Command, args string) error {
	command.SetArgs(strings.Split(args, " "))
	return command.Execute()
}

func ExecuteCliOutErr(command *cobra.Command, args string) (string, string, error) {
	stdOut := os.Stdout
	stdErr := os.Stderr
	r1, w1, err := os.Pipe()
	if err != nil {
		return "", "", err
	}
	r2, w2, err := os.Pipe()
	if err != nil {
		return "", "", err
	}
	os.Stdout = w1
	os.Stderr = w2

	command.SetArgs(strings.Split(args, " "))
	err = command.Execute()

	chan1 := make(chan string)
	chan2 := make(chan string)

	chan1err := make(chan error)
	chan2err := make(chan error)

	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r1)
		chan1err <- err
		chan1 <- buf.String()
	}()
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r2)
		chan2err <- err
		chan2 <- buf.String()
	}()

	// back to normal state
	os.Stdout = stdOut // restoring the real stdout
	os.Stderr = stdErr
	if err := w1.Close(); err != nil {
		return "", "", err
	}
	if err := w2.Close(); err != nil {
		return "", "", err
	}
	if err := <-chan1err; err != nil {
		return "", "", err
	}
	if err := <-chan2err; err != nil {
		return "", "", err
	}
	capturedStdout := <-chan1
	capturedStderr := <-chan2

	return strings.TrimSuffix(capturedStdout, "\n"),
		strings.TrimSuffix(capturedStderr, "\n"),
		err
}
