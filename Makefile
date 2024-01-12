.PHONY : update bin
.PHONY : lint test integration coverage 
.PHONY : clean clean/coverage clean/bin

PROJECT_PATH = $(shell pwd -L)
BINDIR = $(PROJECT_PATH)/.bin
GOFLAGS ::= ${GOFLAGS}
COVERDIR = $(PROJECT_PATH)/.coverage
COVEROUT = $(wildcard $(COVERDIR)/*.out)
COVERINTERCHANGE = $(COVEROUT:.out=.interchange)
COVERHTML = $(COVEROUT:.out=.html)
COVERXML = $(COVEROUT:.out=.xml)
COVERCOMBINED ::= $(COVERDIR)/combined.out
GOIMPORT_LOCAL = github.com/kevinconway/
GOLANGCILINT_CONFIG = $(PROJECT_PATH)/.golangci.yaml

update:
	GO111MODULE=on go get -u

bin: $(BINDIR)

$(BINDIR):
	mkdir -p $(BINDIR)
	GOBIN=$(BINDIR) go install github.com/golang/mock/mockgen@v1.6.0
	GOBIN=$(BINDIR) go install golang.org/x/tools/cmd/goimports@latest
	GOBIN=$(BINDIR) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.52.2
	GOBIN=$(BINDIR) go install github.com/axw/gocov/gocov@v1.1.0 
	GOBIN=$(BINDIR) go install github.com/matm/gocov-html/cmd/gocov-html@v1.4.0
	GOBIN=$(BINDIR) go install github.com/AlekSi/gocov-xml@v1.1.0
	GOBIN=$(BINDIR) go install github.com/wadey/gocovmerge@latest

fmt: $(BINDIR)
	# Apply goimports to all code files. Here we intentionally
	# ignore everything in /vendor if it is present.
	GO111MODULE=on \
	GOFLAGS="$(GOFLAGS)" \
	$(BINDIR)/goimports -w -v \
	-local $(GOIMPORT_LOCAL) \
	$(shell find . -type f -name '*.go' -not -path "./vendor/*")

lint: $(BINDIR)
	GO111MODULE=on \
	GOFLAGS="$(GOFLAGS)" \
	$(BINDIR)/golangci-lint run \
		--config $(GOLANGCILINT_CONFIG) \
		--print-resources-usage \
		--verbose

test: $(BINDIR) $(COVERDIR)
	GO111MODULE=on \
	GOFLAGS="$(GOFLAGS)" \
	CGO_ENABLED=1 \
	go test \
		-v \
		-cover \
		-race \
		-coverprofile="$(COVERDIR)/unit.out" \
		./...

generate: $(BINDIR)
	# Run any code generation steps.
	GO111MODULE=on \
	GOFLAGS="$(GOFLAGS)" \
	PATH="${PATH}:$(BINDIR)" \
	go generate ./...
	$(MAKE) fmt

coverage: $(BINDIR) $(COVERDIR) $(COVERCOMBINED) $(COVERINTERCHANGE) $(COVERHTML) $(COVERXML)
	# The cover rule is an alias for a number of other rules that each
	# generate part of the full coverage report. First, any coverage reports
	# are combined so that there is a report both for an individual test run
	# and a report that covers all test runs together. Then all coverage
	# files are converted to an interchange format. From there we generate
	# an HTML and XML report. XML reports may be used with jUnit style parsers,
	# the HTML report is for human consumption in order to help identify
	# the location of coverage gaps, and the original reports are available
	# for any purpose.
	GO111MODULE=on \
	GOFLAGS="$(GOFLAGS)" \
	go tool cover -func $(COVERCOMBINED)

$(COVERCOMBINED):
	GO111MODULE=on \
	GOFLAGS="$(GOFLAGS)" \
 	$(BINDIR)/gocovmerge $(COVERDIR)/*.out > $(COVERCOMBINED)

	# NOTE: I couldn't figure out how to automatically include
	# the combined files with the list of other .out files that
	# are processed in bulk. For now, this needs to have specific
	# calls to make for combined coverage.
	$(MAKE) $(COVERCOMBINED:.out=.interchange)
	$(MAKE) $(COVERCOMBINED:.out=.xml)
	$(MAKE) $(COVERCOMBINED:.out=.html)

$(COVERDIR)/%.interchange: $(COVERDIR)/%.out
	GO111MODULE=on \
	GOFLAGS="$(GOFLAGS)" \
	$(BINDIR)/gocov convert $< > $@

$(COVERDIR)/%.xml: $(COVERDIR)/%.interchange
	cat $< | \
	GO111MODULE=on \
	GOFLAGS="$(GOFLAGS)" \
	$(BINDIR)/gocov-xml > $@

$(COVERDIR)/%.html: $(COVERDIR)/%.interchange
	cat $< | \
	GO111MODULE=on \
	GOFLAGS="$(GOFLAGS)" \
	$(BINDIR)/gocov-html > $@

$(COVERDIR): 
	mkdir -p $(COVERDIR)

clean: clean/coverage clean/bin ;

clean/bin:
	rm -rf $(BINDIR)

clean/coverage:
	rm -rf $(COVERDIR)
