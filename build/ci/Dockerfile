FROM alpine:3.22

WORKDIR /tmp

# Install dependencies

RUN apk update \
    && apk add --update --no-cache bash curl git make gcc musl-dev docker

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

# Install dependencies

WORKDIR /sdvg

COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum

RUN git config --global --add safe.directory /sdvg \
    && go mod download
