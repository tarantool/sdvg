default: docs
all: default

doc: doc/prepare doc/serve

doc/prepare:
	pip install -r ./doc/build/requirements.txt

doc/serve:
	mkdocs serve \
	  --dev-addr "127.0.0.1:8000" \
	  --open \
	  --strict \
	  --clean \
	  -f ./mkdocs.yml

.PHONY: doc/build
doc/build:
	mkdocs build \
	  --strict \
	  --site-dir ./site
