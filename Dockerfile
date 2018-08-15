FROM golang:1.10

ADD . /go/src/git.shadow53.com/BluestNight/nebula-forms

RUN go install git.shadow53.com/BluestNight/nebula-forms
RUN mkdir -p /go/plugins
RUN go build -buildmode=plugin -o /usr/lib/nebula-forms/plugins/email.so git.shadow53.com/BluestNight/nebula-forms/plugins/email

ENV CONFIG_FILE=/etc/nebula-forms/config.toml

WORKDIR /

EXPOSE 2002

VOLUME /etc/nebula-forms

ENTRYPOINT /go/bin/nebula-forms -conf $CONFIG_FILE
