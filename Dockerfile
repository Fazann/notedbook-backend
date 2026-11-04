# API image only: docker build -t notebook-api .
# No BuildKit-only syntax (RUN --mount), so it also builds with the legacy builder (e.g. Cloud Build).
# Swagger is local-only: the image is always built with -tags noswagger, so docs/ is not needed here.

# ---- build: static, stripped API binary ----
FROM golang:1.26-alpine AS build
WORKDIR /src

# Download modules in their own layer so code changes do not redo it.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# noswagger leaves out the Swagger UI and the generated docs/ package.
# timetzdata embeds the zone database because the final image has none.
ENV CGO_ENABLED=0 \
    GOFLAGS="-trimpath -buildvcs=false"
RUN go build -tags timetzdata,noswagger -ldflags="-s -w" -o /out/app ./cmd/app

# ---- runtime: empty image with CA certificates (S3, SMTP) and the binary ----
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/app /app/app

ENV GIN_MODE=release \
    PORT=3010
EXPOSE 3010

# 65532 is the conventional "nonroot" uid/gid; no shell or passwd file is needed.
USER 65532:65532
ENTRYPOINT ["/app/app"]
