FROM golang:1.10

ADD . /go/src/github.com/BluestNight/static-forms

RUN go install github.com/BluestNight/static-forms

ENV CONFIG_FILE=/etc/static-forms/config.toml

WORKDIR /

EXPOSE 2002

VOLUME /etc/static-forms

ENTRYPOINT /go/bin/static-forms -conf $CONFIG_FILE
