#----------------------------------------------------------------------------------
# Release
#----------------------------------------------------------------------------------

.PHONY: upload-github-release-assets
upload-github-release-assets: render-yaml
	go run ci/upload_github_release_assets.go

.PHONY: release
release: docker-push upload-github-release-assets
