# Arguments

ci_image = ghcr.io/hoodiehuuuuu/sdvg-ci:0.0.2

# Targets

default: ci
all: default

ci: ci/build ci/push

ci/build:
	docker build \
      -f ./build/ci/Dockerfile \
      --platform linux/amd64 \
      --tag $(ci_image) .

ci/push:
	docker push $(ci_image)
