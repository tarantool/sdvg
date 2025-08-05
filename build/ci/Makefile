# Arguments

ci_image = ghcr.io/hoodiehuuuuu/sdvg-ci:0.0.1

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

bench/prepare:
	pip install -r ./build/ci/requirements.txt

old_bench ?= benchmark-master.txt
new_bench ?= benchmark-pr.txt
alert_threshold ?= 5
aggregation_method ?= "mean"

bench/compare:
	python ./build/ci/compare_benchmarks.py \
	  $(old_bench) $(new_bench) \
	  --alert-threshold $(alert_threshold) \
	  --aggregation $(aggregation_method)