FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=secret,id=cloud_ca,target=/tmp/cloud-ca.pem \
    if [ -f /tmp/cloud-ca.pem ]; then export SSL_CERT_FILE=/tmp/cloud-ca.pem; fi; \
    go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/server .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
