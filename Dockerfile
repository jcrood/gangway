FROM golang:1.14.2-stretch AS buildstage
WORKDIR /go/src/github.com/jcrood/gangway

RUN go get -u github.com/mjibson/esc/...
COPY . .
RUN esc -o cmd/gangway/bindata.go templates/

ENV GO111MODULE on
RUN go mod verify
RUN CGO_ENABLED=0 GOOS=linux go install -ldflags="-w -s" -v github.com/jcrood/gangway/...


FROM gcr.io/distroless/static:nonroot
USER 1001:1001
COPY --from=buildstage /go/bin/gangway /bin/gangway
