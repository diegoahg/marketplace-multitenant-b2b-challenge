FROM golang:1.25-bookworm
RUN go install github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0 \
    && go install github.com/securego/gosec/v2/cmd/gosec@v2.29.0 \
    && go install github.com/rhysd/actionlint/cmd/actionlint@v1.7.7
WORKDIR /repo
