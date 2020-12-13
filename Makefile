NAMESPACE	= akeil.net/akeil
NAME		= rmtool
QNAME		= $(NAMESPACE)/$(NAME)
BINDIR		= ./bin


build:
	mkdir -p $(BINDIR)

test:
	go test $(QNAME) $(QNAME)

src = $(wildcard *.go) $(wildcard ./*/*/*.go)

fmt: ${src}
	for file in $^ ; do\
		gofmt -w $${file} ;\
	done
