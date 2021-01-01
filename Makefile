NAMESPACE	:= akeil.net/akeil
NAME		:= rm
QNAME		:= $(NAMESPACE)/$(NAME)
BINDIR		:= ./bin
EXAMPLESDIR := $(BINDIR)/examples
BRUSHDIR    := data/brushes
SPRITES     := data/sprites

SRC := $(wildcard *.go) $(wildcard ./*/*.go) $(wildcard ./*/*/*.go)
BRUSHES := $(wildcard $BRUSHDIR/*.png)

.PHONY: examples

all: build cli examples sprites

build:
	go build

cli:
	mkdir -p $(BINDIR)
	go build -o $(BINDIR)/rmtool cmd/rmtool/main.go

examples: ${samples}
	mkdir -p $(EXAMPLESDIR)
	go build -o $(EXAMPLESDIR)/api examples/api/main.go
	go build -o $(EXAMPLESDIR)/browse examples/browse/main.go
	go build -o $(EXAMPLESDIR)/render examples/render/main.go

test:
	go test $(QNAME) $(QNAME)

fmt: ${SRC}
	for file in $^ ; do\
		gofmt -w $${file} ;\
	done

lint: ${SRC}
	for file in $^ ; do\
		golint -min_confidence 0.6 $${file} ;\
	done

sprites: $(BRUSHES)
	go run scripts/mksprite.go $(BRUSHDIR) $(SPRITES)
