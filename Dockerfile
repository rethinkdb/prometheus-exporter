FROM golang:1.14.0 as build

COPY . /src
RUN set -ex \
    && cd /src \
    && CGO_ENABLED=0 go build -o /bin/prometheus-exporter

FROM alpine:latest
MAINTAINER CMogilko <cmogilko@gmail.com>

COPY --from=build /bin/prometheus-exporter /bin/prometheus-exporter

USER nobody
EXPOSE     9055
ENTRYPOINT [ "/bin/prometheus-exporter" ]
