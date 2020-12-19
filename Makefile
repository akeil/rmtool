NAMESPACE	= akeil.net/akeil
NAME		= rm
QNAME		= $(NAMESPACE)/$(NAME)
BINDIR		= ./bin
EXAMPLESDIR = $(BINDIR)/examples

.PHONY: examples

build:
	go build

examples: ${samples}
	mkdir -p $(EXAMPLESDIR)
	go build -o $(EXAMPLESDIR)/api examples/api/main.go
	go build -o $(EXAMPLESDIR)/browse examples/browse/main.go
	go build -o $(EXAMPLESDIR)/render examples/render/main.go

test:
	go test $(QNAME) $(QNAME)

src = $(wildcard *.go) $(wildcard ./*/*/*.go)

fmt: ${src}
	for file in $^ ; do\
		gofmt -w $${file} ;\
	done
