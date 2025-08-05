default: docs
all: default

docs: docs/prepare docs/serve

docs/prepare:
	pip install -r ./doc/common/requirements.txt

docs/serve:
	mkdocs serve \
      --dev-addr "127.0.0.1:8000" \
      --open \
      --strict \
      --clean \
      -f ./mkdocs.yml