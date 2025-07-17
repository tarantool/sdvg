FROM --platform=${BUILDPLATFORM} golang:1.23.8-alpine AS build

RUN apk add --no-cache make

WORKDIR /sdvg/

COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum

RUN go mod download

COPY . .

ARG OS_FAMILY
ARG ARCHITECTURE

RUN make build/binary os_family=${OS_FAMILY} architecture=${ARCHITECTURE}

FROM alpine:3.18.4

ARG OS_FAMILY
ARG ARCHITECTURE

COPY --from=build /sdvg/build/out/sdvg-${OS_FAMILY}-${ARCHITECTURE} /bin/sdvg

ENTRYPOINT ["/bin/sdvg"]
