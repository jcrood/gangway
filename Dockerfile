FROM golang:1.18.2-bullseye AS buildstage
WORKDIR /go/src/github.com/jcrood/gangway

COPY . .

RUN go mod verify
RUN CGO_ENABLED=0 GOOS=linux go install -ldflags="-w -s" -v github.com/jcrood/gangway/...


FROM gcr.io/distroless/static:nonroot
USER 1001:1001
COPY --from=buildstage /go/bin/gangway /bin/gangway
