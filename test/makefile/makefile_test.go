package install

import (
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/go-utils/errors"
)

// The makefile prints its configurable values in the form:
/*
make print_configuration
echo "$MAKE_CONFIGURATION"
Build state
 phase: "dev"
Images configuration
 repo:
 org: soloio
 tag: 15045455-dev
 gcloud_project_id:
 full_spec: soloio
 sample: soloio/<container_name>:15045455-dev
*/
// Use this make target to validate the configuration for a given set of inputs

var _ = Describe("verify that the Makefile's configurable variables take expected values", func() {

	It("should work in the dev lifecycle phase, with defaults", func() {
		envVars := []string{}
		config, err := callMakeDebugTarget(envVars)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		expectPhase(config, "dev")
		expectRepo(config, "")
		expectOrg(config, "soloio")
		expectTag(config, "[0-9]{8}-dev")
		expectGcloudProjectId(config, "")
		expectFullSpec(config, "soloio")
		expectSample(config, "soloio/<container_name>:[0-9]{8}-dev")
	})

	It("should work in the dev lifecycle phase, with overrides", func() {
		containerRepo := "quay.io"
		containerOrg := "gloocorp"
		imageTag := "latesttag"
		envVars := []string{
			makeEnvVar("CONTAINER_REPO", containerRepo),
			makeEnvVar("CONTAINER_ORG", containerOrg),
			makeEnvVar("IMAGE_TAG", imageTag),
		}
		config, err := callMakeDebugTarget(envVars)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		expectPhase(config, "dev")
		expectRepo(config, containerRepo)
		expectOrg(config, containerOrg)
		expectTag(config, imageTag)
		expectGcloudProjectId(config, "")
		expectFullSpec(config, fmt.Sprintf("%s/%s", containerRepo, containerOrg))
		expectSample(config, fmt.Sprintf("%s/%s/<container_name>:%s",
			containerRepo,
			containerOrg,
			imageTag))
	})

	It("should work in the buildtest lifecycle phase", func() {
		gcpId := "someproject"
		envVars := []string{
			makeEnvVar("GCLOUD_PROJECT_ID", gcpId),
		}
		config, err := callMakeDebugTarget(envVars)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		expectPhase(config, "buildtest")
		expectRepo(config, fmt.Sprintf("gcr.io/%s", gcpId))
		expectOrg(config, "soloio")
		expectTag(config, "[a-z0-9]{6}-buildtest")
		expectGcloudProjectId(config, gcpId)
		expectFullSpec(config, fmt.Sprintf("gcr.io/%s/soloio", gcpId))
		expectSample(config, fmt.Sprintf("gcr.io/%s/soloio/<container_name>:[a-z0-9]{6}-buildtest", gcpId))
	})

	It("should work in the release lifecycle phase", func() {
		taggedVersion := "v1.2.3"
		version := "1.2.3"
		envVars := []string{
			makeEnvVar("TAGGED_VERSION", taggedVersion),
		}
		config, err := callMakeDebugTarget(envVars)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		expectPhase(config, "release")
		expectRepo(config, "")
		expectOrg(config, "soloio")
		expectTag(config, version)
		expectGcloudProjectId(config, "")
		expectFullSpec(config, "soloio")
		expectSample(config, fmt.Sprintf("soloio/<container_name>:%s", version))
	})
})

func expectPhase(config, phase string) {
	ExpectWithOffset(2, config).To(MatchRegexp(fmt.Sprintf("phase: \"%s\"", phase)))
}
func expectRepo(config, val string) {
	if val == "" {
		ExpectWithOffset(2, strings.Replace(config, "\n", "", -1)).
			To(MatchRegexp(fmt.Sprintf("repo:  org:")))
		return
	}
	ExpectWithOffset(2, config).To(MatchRegexp(fmt.Sprintf("repo: %s", val)))
}
func expectOrg(config, val string) {
	ExpectWithOffset(2, config).To(MatchRegexp(fmt.Sprintf("org: %s", val)))
}
func expectTag(config, val string) {
	ExpectWithOffset(2, config).To(MatchRegexp(fmt.Sprintf("tag: %s", val)))
}
func expectGcloudProjectId(config, val string) {
	if val == "" {
		ExpectWithOffset(2, strings.Replace(config, "\n", "", -1)).
			To(MatchRegexp(fmt.Sprintf("gcloud_project_id:  full_spec:")))
		return
	}
	ExpectWithOffset(2, config).To(MatchRegexp(fmt.Sprintf("gcloud_project_id: %s", val)))
}
func expectFullSpec(config, val string) {
	ExpectWithOffset(2, config).To(MatchRegexp(fmt.Sprintf("full_spec: %s", val)))
}
func expectSample(config, val string) {
	ExpectWithOffset(2, config).To(MatchRegexp(fmt.Sprintf("sample: %s", val)))
}

func callMakeDebugTarget(envVars []string) (string, error) {
	cmd := exec.Command("make", "print_configuration")
	cmd.Env = envVars
	cmd.Dir = "../.."
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "error with command, response:\n%s", string(output))
	}
	return string(output), nil
}

// makeEnvVar produces strings in the format expected by exec.Cmd.Env
func makeEnvVar(key, val string) string {
	return fmt.Sprintf("%s=%s", key, val)
}
