# System variables

git_branch := $(if $(CI_BRANCH_NAME),$(CI_BRANCH_NAME),$(shell git branch --show-current 2>/dev/null))

git_last_tag := $(shell git describe --tags --exclude=latest --abbrev=0 2>/dev/null || echo 0.0.0)

app_version := $(shell git describe --tags --exclude=latest --abbrev=8 2>/dev/null || echo 0.0.0)
ifneq ($(git_branch),master) # not on master branch
	app_version := $(app_version:$(git_last_tag)-%=$(git_last_tag)-$(git_branch)-%)
endif

release_app_version := $(app_version)

race =
build_flags = -ldflags="-X main.version=$(app_version)" $(if $(race),--race,)
os_family = linux
architecture = amd64

# Targets

build: build/binary build/docker

build/binary:
	CGO_ENABLED=0 GOOS=$(os_family) GOARCH=$(architecture) go build $(build_flags) \
		-o "./build/out/sdvg-$(os_family)-$(architecture)" ./cmd/sdvg/main.go

build/docker:
	docker build -f ./build/package/Dockerfile --tag sdvg-$(os_family)-$(architecture):$(app_version) \
        --platform $(os_family)/$(architecture) \
		--build-arg OS_FAMILY=$(os_family) \
		--build-arg ARCHITECTURE=$(architecture) .
	mkdir -p ./build/out/
	docker image save -o "./build/out/sdvg-$(os_family)-$(architecture)-image.tar.gz" \
		"sdvg-$(os_family)-$(architecture):$(app_version)"
