ARG GO_VERSION=1.15
ARG ALPINE_VERSION=3.12

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

ENV GO111MODULE on
ENV GOSUMDB off

WORKDIR /go/src/actuary

COPY . .
RUN go install ./cmd/actuary/

FROM alpine:${ALPINE_VERSION}

COPY --from=builder /go/bin/actuary /usr/bin/actuary

USER 32256:32256

ENTRYPOINT [ "/usr/bin/actuary" ]
