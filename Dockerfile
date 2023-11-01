FROM golang:1.21.3-alpine3.18 AS buildstage

WORKDIR /go/src/github.com/jcrood/gangway

ADD https://raw.githubusercontent.com/Dogfalo/materialize/v1-dev/dist/css/materialize.min.css assets/
ADD https://raw.githubusercontent.com/Dogfalo/materialize/v1-dev/dist/js/materialize.min.js assets/
ADD https://raw.githubusercontent.com/PrismJS/prism/v1.28.0/components/prism-core.min.js assets/
ADD https://raw.githubusercontent.com/PrismJS/prism/v1.28.0/components/prism-bash.min.js assets/
ADD https://raw.githubusercontent.com/PrismJS/prism/v1.28.0/components/prism-yaml.min.js assets/
ADD https://raw.githubusercontent.com/PrismJS/prism/v1.28.0/components/prism-powershell.min.js assets/
ADD https://raw.githubusercontent.com/PrismJS/prism/v1.28.0/themes/prism-tomorrow.min.css assets/

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY assets/ assets/
COPY cmd/ cmd/
COPY internal/ internal/
COPY templates/ templates/

RUN cd cmd/gangway && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/gangway


FROM gcr.io/distroless/static:nonroot

USER 1001:1001
COPY --from=buildstage /go/bin/gangway /bin/gangway
