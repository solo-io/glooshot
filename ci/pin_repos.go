package main

import (
	"log"

	version "github.com/solo-io/go-utils/versionutils"
)

func main() {
	tomlTree, err := version.ParseToml()
	fatalCheck(err, "parsing error")

	soloKitVersion, err := version.GetVersion(version.SoloKitPkg, tomlTree)
	fatalCheck(err, "getting solo-kit version")

	glooVersion, err := version.GetVersion(version.GlooPkg, tomlTree)
	fatalCheck(err, "getting gloo version")

	superglooVersion, err := version.GetVersion(version.SuperglooPkg, tomlTree)
	fatalCheck(err, "getting supergloo version")

	fatalCheck(version.PinGitVersion("../solo-kit", soloKitVersion), "consider git fetching in solo-kit repo")

	fatalCheck(version.PinGitVersion("../gloo", glooVersion), "consider git fetching in gloo repo")

	fatalCheck(version.PinGitVersion("../supergloo", superglooVersion), "consider git fetching in supergloo repo")
}

func fatalCheck(err error, msg string) {
	if err != nil {
		log.Fatalf("Error (%v) unable to pin repos!: %v", msg, err)
	}
}
