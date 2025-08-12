FROM alpine:3.22

WORKDIR /tmp

# Install dependencies

RUN apk update \
    && apk add --update --no-cache bash curl git make gcc musl-dev docker tar

# Configure python

RUN apk add --no-cache python3 py3-pip \
    && python3 -m venv /venv \
    && /venv/bin/pip install --upgrade pip setuptools wheel

ENV PATH="/venv/bin:$PATH"

# Configure Go

ARG GOLANG_VERSION=1.23.8

ENV PATH=$PATH:/root/go/bin:/usr/local/go/bin
ENV CGO_ENABLED=1

RUN curl -Lo go.tar.gz https://go.dev/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go.tar.gz \
    && rm go.tar.gz \
    && go version

# Install golangci-lint

ARG GOLANGCI_LINT_VERSION=1.62.2

RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@v${GOLANGCI_LINT_VERSION} \
    && golangci-lint --version

# Install ghub CLI

ARG GH_CLI_VERSION=2.76.2

RUN curl -fsSL -o gh.tar.gz \
      "https://github.com/cli/cli/releases/download/v${GH_CLI_VERSION}/gh_${GH_CLI_VERSION}_linux_amd64.tar.gz" \
    && tar -xzf gh.tar.gz \
    && mv gh_${GH_CLI_VERSION}_linux_amd64/bin/gh /usr/local/bin/gh \
    && rm -rf gh_${GH_CLI_VERSION}_linux_amd64 gh.tar.gz

# Install dependencies

WORKDIR /sdvg

COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY ./build/ci/requirements.txt ./requirements.txt

RUN git config --global --add safe.directory /sdvg \
    && go mod download \
    && pip install -r ./requirements.txt
