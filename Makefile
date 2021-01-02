NAMESPACE	:= github.com/akeil
NAME		:= rm
QNAME		:= $(NAMESPACE)/$(NAME)
BINDIR		:= ./bin
BRUSHDIR    := data/brushes
SPRITES     := data/sprites

SRC := $(wildcard *.go) $(wildcard ./*/*.go) $(wildcard ./*/*/*.go)
BRUSHES := $(wildcard $BRUSHDIR/*.png)

.PHONY: examples

all: build cli sprites

build:
	go build

cli:
	mkdir -p $(BINDIR)
	go build -o $(BINDIR)/rmtool cmd/rmtool/*

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
