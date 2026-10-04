#!/usr/bin/env bash
# Build and run the API image locally.
#
#   scripts/docker.sh build      build the API image
#   scripts/docker.sh run        build, then start the API in the background on $PORT (default 3010)
#   scripts/docker.sh logs       follow the API logs
#   scripts/docker.sh stop       stop and remove the API container
#
# Config comes from .env (or ENV_FILE=path). localhost / 127.0.0.1 in DB_HOST and REDIS_ADDR is
# replaced by host.docker.internal, so the container reaches Postgres and Redis on this machine.
# SWAGGER=true includes the Swagger UI (/docs/index.html) in the API image.
# IMAGE, CONTAINER, PLATFORM and HOST_PORT (port on this machine, default $PORT) can be overridden the
# same way, e.g. HOST_PORT=3011 scripts/docker.sh run when `make run` already uses 3010.
set -euo pipefail

cd "$(dirname "$0")/.."

IMAGE="${IMAGE:-notebook-api:latest}"
SWAGGER="${SWAGGER:-false}"
CONTAINER="${CONTAINER:-notebook-api}"
ENV_FILE="${ENV_FILE:-.env}"
PLATFORM="${PLATFORM:-}"

die() {
	echo "error: $*" >&2
	exit 1
}

# Value of key in the env file, without surrounding quotes; empty when missing.
env_value() {
	local line
	line="$(grep -E "^$1=" "$ENV_FILE" | tail -n 1 || true)"
	line="${line#*=}"
	line="${line%\"}"
	line="${line#\"}"
	echo "${line%\'}" | sed "s/^'//"
}

to_docker_host() {
	echo "$1" | sed -E 's/^(localhost|127\.0\.0\.1)/host.docker.internal/'
}

# Arguments for `docker run`: the env file and the host overrides.
run_args() {
	[[ -f "$ENV_FILE" ]] || die "$ENV_FILE not found; copy .env.example and fill it in"

	RUN_ARGS=(--env-file "$ENV_FILE" --add-host=host.docker.internal:host-gateway)
	[[ -n "$PLATFORM" ]] && RUN_ARGS+=(--platform "$PLATFORM")

	local db_host redis_addr
	db_host="$(env_value DB_HOST)"
	redis_addr="$(env_value REDIS_ADDR)"
	[[ -n "$db_host" ]] && RUN_ARGS+=(-e "DB_HOST=$(to_docker_host "$db_host")")
	[[ -n "$redis_addr" ]] && RUN_ARGS+=(-e "REDIS_ADDR=$(to_docker_host "$redis_addr")")
	return 0
}

build() {
	local args=(-t "$IMAGE" --build-arg "SWAGGER=$SWAGGER")
	[[ -n "$PLATFORM" ]] && args+=(--platform "$PLATFORM")
	docker build "${args[@]}" .
}

run() {
	build
	run_args
	local port
	port="$(env_value PORT)"
	port="${port:-3010}"
	local host_port="${HOST_PORT:-$port}"

	docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
	docker run -d --name "$CONTAINER" --restart unless-stopped \
		-p "$host_port:$port" -e "PORT=$port" "${RUN_ARGS[@]}" "$IMAGE" >/dev/null

	echo "API running in container '$CONTAINER' on http://localhost:$host_port"
	echo "logs: $0 logs    stop: $0 stop"
}

case "${1:-}" in
build) build ;;
run) run ;;
logs) docker logs -f "$CONTAINER" ;;
stop) docker rm -f "$CONTAINER" >/dev/null && echo "stopped $CONTAINER" ;;
*)
	sed -n '2,13p' "$0" | sed 's/^# \{0,1\}//'
	exit 1
	;;
esac
