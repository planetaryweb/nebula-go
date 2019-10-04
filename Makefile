#.ifeq ($(OS),Windows_NT)
#    CCFLAGS += -D WIN32
#    .ifeq ($(PROCESSOR_ARCHITECTURE),AMD64)
#        CCFLAGS += -D AMD64
#    .endif
#    .ifeq ($(PROCESSOR_ARCHITECTURE),x86)
#        CCFLAGS += -D IA32
#    .endif
#.else
#    UNAME_S := $(shell uname -s)
#    .ifeq ($(UNAME_S),Linux):
#		PREFIX ?= /usr/local
#    .endif
#    .ifeq ($(UNAME_S),Darwin)
#        CCFLAGS += -D OSX
#    .endif
#.endif

WD := .PARSEDIR

all: grpc proto build-nebula build-plugins install-nebula install-plugins

build: build-nebula build-plugins

build-nebula:
	go build -o nebula-forms

build-plugins: build-plugin-email

build-plugin-email:
	go build -o nebula-email ./plugins/email

clean: clean-nebula clean-plugins

clean-nebula:
	rm -f ./nebula-forms

clean-plugins: clean-plugin-email

clean-plugin-email:
	rm -f ./nebula-email

clean-proto:
	rm -f proto/handler.pb.go

grpc:
	go install google.golang.org/grpc

install: install-nebula install-plugins

install-nebula: build-nebula
	install -C -D $PREFIX/bin ./nebula-forms

install-plugins: install-plugin-email

install-plugin-email: build-plugin-email
	install -C -D $PREFIX/bin ./nebula-email

proto! proto/handler.proto
	protoc -I proto/ proto/handler.proto --go_out=plugins=grpc:proto
