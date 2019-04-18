package install

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"

	"github.com/solo-io/go-utils/contextutils"
	"go.uber.org/zap"

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
			cliOut, err := glooshotWithLogger("--h")
			Expect(err).To(HaveOccurred())
			Expect(cliOut.CobraStdout).To(Equal(""))
			Expect(cliOut.CobraStderr).To(standardCobraHelpBlockMatcher)
			Expect(cliOut.LoggerConsoleStout).To(Equal(""))
			// Assert the intention with regexes
			Expect(cliOut.LoggerConsoleStderr).To(MatchRegexp("unknown flag: --h"))
			Expect(cliOut.LoggerConsoleStderr).To(MatchRegexp(cli.ErrorMessagePreamble))
			// Assert the details for documentation purposes (flake-prone)
			Expect(cliOut.LoggerConsoleStderr).To(Equal(`error during glooshot cli execution	{"version": "dev", "error": "unknown flag: --h"}

`))
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
	mockTargets := cli.NewMockTargets()
	testCliLogger := cli.BuildMockedCliLogger([]string{".glooshot", "log"}, cli.OutputModeEnvVar, &mockTargets)
	ctx := cli.GetInitialContextAndSetLogger(testCliLogger)
	app := cli.App(ctx, "testglooshotcli")
	cStdout, cStderr, err := ExecuteCliOutErr(ctx, app, args)
	return cStdout, cStderr, err
}

func glooshotWithLogger(args string) (CliOutput, error) {
	mockTargets := cli.NewMockTargets()
	testCliLogger := cli.BuildMockedCliLogger([]string{".glooshot", "log"}, cli.OutputModeEnvVar, &mockTargets)
	ctx := cli.GetInitialContextAndSetLogger(testCliLogger)
	app := cli.App(ctx, "testglooshotcli")
	cliOut := CliOutput{}
	var err error
	cliOut.CobraStdout, cliOut.CobraStderr, err = ExecuteCliOutErr(ctx, app, args)
	// After the command has been executed, there should be content in the logs
	cliOut.LoggerConsoleStout, _, _ = mockTargets.Stdout.Summarize()
	cliOut.LoggerConsoleStderr, _, _ = mockTargets.Stderr.Summarize()
	return cliOut, err
}

////////////////////////////////////////////////////////////////////////////////
// TODO(mitchdraft) replace with https://github.com/solo-io/go-utils/pull/125 on merge
////////////////////////////////////////////////////////////////////////////////
// CliOutput captures all the relevant output from a Cobra Command
// For clarity and simplicity, output from zapcore loggers are stored separately
// otherwise, it would be neccessary to coordinate the initialization of the loggers
// with the os.Std*** manipulation done in ExecuteCliOutErr
type CliOutput struct {
	LoggerConsoleStderr string
	LoggerConsoleStout  string
	CobraStderr         string
	CobraStdout         string
}

func ExecuteCli(command *cobra.Command, args string) error {
	command.SetArgs(strings.Split(args, " "))
	return command.Execute()
}

func ExecuteCliOutErr(ctx context.Context, command *cobra.Command, args string) (string, string, error) {
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
	if err != nil {
		contextutils.LoggerFrom(ctx).Errorw("error during glooshot cli execution", zap.Error(err))
	}

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
	//capturedStdout := ""
	//for len(chan1) > 0 {
	//	capturedStdout += <-chan1
	//}
	//capturedStderr := ""
	//for len(chan2) > 0 {
	//	capturedStderr += <-chan2
	//}

	return strings.TrimSuffix(capturedStdout, "\n"),
		strings.TrimSuffix(capturedStderr, "\n"),
		err
}
