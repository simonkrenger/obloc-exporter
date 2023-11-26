FROM registry.fedoraproject.org/fedora-minimal:latest as build
WORKDIR /go/src/gitlab.com/simonkrenger/obloc-exporter
COPY . .
RUN microdnf install -y golang git && go get
# http://blog.wrouesnel.com/articles/Totally%20static%20Go%20builds/
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' main.go

FROM scratch
LABEL maintainer="Simon Krenger <simon@krenger.ch>"
WORKDIR /
COPY --from=0 /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem /etc/ssl/certs/
COPY --from=0 /go/src/gitlab.com/simonkrenger/obloc-exporter/main ./obloc-exporter

EXPOSE 8081
USER 1001
CMD ["./obloc-exporter"]
