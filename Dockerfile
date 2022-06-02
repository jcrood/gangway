FROM golang:1.18.2-bullseye AS buildstage
WORKDIR /go/src/github.com/jcrood/gangway

ADD https://raw.githubusercontent.com/PrismJS/prism/v1.28.0/components/prism-core.min.js assets/
ADD https://raw.githubusercontent.com/PrismJS/prism/v1.28.0/components/prism-bash.min.js assets/
ADD https://raw.githubusercontent.com/PrismJS/prism/v1.28.0/components/prism-yaml.min.js assets/
ADD https://raw.githubusercontent.com/PrismJS/prism/v1.28.0/components/prism-powershell.min.js assets/
ADD https://raw.githubusercontent.com/PrismJS/prism/v1.28.0/themes/prism-tomorrow.min.css assets/
ADD https://raw.githubusercontent.com/Dogfalo/materialize/v1-dev/dist/css/materialize.min.css assets/
ADD https://raw.githubusercontent.com/Dogfalo/materialize/v1-dev/dist/js/materialize.min.js assets/

COPY . .

RUN go mod verify
RUN CGO_ENABLED=0 GOOS=linux go install -ldflags="-w -s" -v github.com/jcrood/gangway/...


FROM gcr.io/distroless/static:nonroot
USER 1001:1001
COPY --from=buildstage /go/bin/gangway /bin/gangway
