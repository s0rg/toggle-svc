FROM golang:1.15-buster AS builder

ARG BUILD_BASE

ADD . /go/src/$BUILD_BASE
WORKDIR /go/src/$BUILD_BASE

RUN apt-get update -y \
 && apt-get install -y ca-certificates tzdata unzip \
 && apt-get clean \
 && update-ca-certificates --fresh

RUN make mod
RUN make release

RUN mkdir /apps && cp /go/src/$BUILD_BASE/bin/* /apps

FROM scratch

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /apps/toggle-svc /app.bin

USER nobody:nogroup

ENTRYPOINT ["/app.bin"]
