FROM alpine:latest
MAINTAINER CMogilko <cmogilko@gmail.com>

RUN apk add --update -t build-deps go git make

COPY . /src

RUN cd /src && go build -o /bin/prometheus-exporter
RUN rm -rf /src

RUN apk del --purge build-deps go git make

EXPOSE     9055
ENTRYPOINT [ "/bin/prometheus-exporter" ]
