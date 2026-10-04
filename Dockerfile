# syntax=docker/dockerfile:1

# API image only: docker build -t notebook-api .
# Swagger UI is left out by default (~9 MB); add it with --build-arg SWAGGER=true (served when ENVIRONMENT != production).

# ---- build: static, stripped API binary ----
FROM golang:1.26-alpine AS build
WORKDIR /src

# Download modules in their own layer so code changes do not redo it.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG SWAGGER=false
# Same version as `make swagger`; docs/ is git-ignored, so it is generated here when needed.
ARG SWAG_VERSION=v1.16.6
# timetzdata embeds the zone database because the final image has none.
ENV CGO_ENABLED=0 \
    GOFLAGS="-trimpath -buildvcs=false"
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    if [ "$SWAGGER" = "true" ]; then \
        go run github.com/swaggo/swag/cmd/swag@${SWAG_VERSION} init -g cmd/app/main.go -o docs --outputTypes go && \
        TAGS=timetzdata; \
    else \
        TAGS=timetzdata,noswagger; \
    fi && \
    go build -tags "$TAGS" -ldflags="-s -w" -o /out/app ./cmd/app

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
