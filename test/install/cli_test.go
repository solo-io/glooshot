package install

import (
	"github.com/solo-io/go-utils/contextutils"
	"go.uber.org/zap"

	"github.com/solo-io/glooshot/pkg/cli"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/gloo/projects/gloo/cli/pkg/helpers"
	clitestutils "github.com/solo-io/glooshot/test/pregoutils-clitestutils"
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
			Expect(stdErr).To(Equal(""))
			Expect(out).To(Equal(noResourcesTable))

			out, stdErr, err = glooshot("create experiment -f ../../examples/gs_delay.yaml")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).To(Equal(""))
			Expect(out).To(Equal(""))

			out, stdErr, err = glooshot("get experiments --all-namespaces")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).To(Equal(""))
			Expect(out).NotTo(Equal(noResourcesTable))

			out, stdErr, err = glooshot("delete experiments -n default --all")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).To(Equal(""))
			Expect(out).NotTo(Equal(noResourcesTable))

			out, stdErr, err = glooshot("get experiments --all-namespaces")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdErr).To(Equal(""))
			Expect(out).To(Equal(noResourcesTable))
		})
	})

	Context("expect human-friendly errors", func() {

		It("should return human-friendly errors on bad input", func() {
			cliOut := glooshotWithLogger("--h")
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

	})
})

func glooshot(args string) (string, string, error) {
	mockTargets := cli.NewMockTargets()
	testCliLogger := cli.BuildMockedCliLogger([]string{".glooshot", "log"}, cli.OutputModeEnvVar, &mockTargets)
	ctx := cli.GetInitialContextAndSetLogger(testCliLogger)
	app := cli.App(ctx, "testglooshotcli")
	cStdout, cStderr, err := clitestutils.ExecuteCliOutErr(app, args, nil)
	return cStdout, cStderr, err
}

func glooshotWithLogger(args string) clitestutils.CliOutput {
	mockTargets := cli.NewMockTargets()
	testCliLogger := cli.BuildMockedCliLogger([]string{".glooshot", "log"}, cli.OutputModeEnvVar, &mockTargets)
	ctx := cli.GetInitialContextAndSetLogger(testCliLogger)
	app := cli.App(ctx, "testglooshotcli")
	cliOut := clitestutils.CliOutput{}
	var err error
	commandErrorHandler := func(commandErr error) {
		if commandErr != nil {
			contextutils.LoggerFrom(ctx).Errorw(cli.ErrorMessagePreamble, zap.Error(commandErr))
		}
	}
	cliOut.CobraStdout, cliOut.CobraStderr, err = clitestutils.ExecuteCliOutErr(app, args, commandErrorHandler)
	Expect(err).NotTo(HaveOccurred())
	// After the command has been executed, there should be content in the logs
	cliOut.LoggerConsoleStout, _, _ = mockTargets.Stdout.Summarize()
	cliOut.LoggerConsoleStderr, _, _ = mockTargets.Stderr.Summarize()
	return cliOut
}
