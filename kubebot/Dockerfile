FROM golang:1.21 as builder

WORKDIR /go/src/app
COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN make build TARGETOS=${TARGETOS} TARGETARCH=${TARGETARCH}

FROM scratch

WORKDIR /
COPY --from=builder /go/src/app/slackbot .
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 9090

ENTRYPOINT [ "./slackbot", "start" ]