IMAGE ?= squirrelbot:0.1.0
CONTAINER ?= $(shell command -v docker >/dev/null 2>&1 && echo docker || echo podman)
COMPOSE = $(CONTAINER) compose

.DEFAULT_GOAL := help

.PHONY: help dev bot web db up down build build-bot build-web image test test-e2e

help:
	@printf '%s\n' \
		'dev          Postgres, Air, and the dashboard' \
		'bot          Air only' \
		'web          Dashboard dev server' \
		'db           Postgres via $(CONTAINER)' \
		'up           Bot and Postgres via $(CONTAINER)' \
		'down         Stop $(CONTAINER) compose services' \
		'build        Local bot binary and dashboard' \
		'image        Container image ($(IMAGE))' \
		'test         Go unit tests' \
		'test-e2e     API and dashboard end-to-end tests'

dev: db web/node_modules
	@test -f .env || { echo 'Copy .env.example to .env and fill it in.' >&2; exit 1; }
	@trap 'kill 0' INT TERM EXIT; \
		npm --prefix web run dev & \
		"$$(go -C tools tool -n air)" & \
		wait

bot:
	@test -f .env || { echo 'Copy .env.example to .env and fill it in.' >&2; exit 1; }
	@exec "$$(go -C tools tool -n air)"

web: web/node_modules
	npm --prefix web run dev

db:
	$(COMPOSE) up postgres -d

up:
	$(COMPOSE) up --build

down:
	$(COMPOSE) down

build: build-bot build-web

build-bot:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/squirrelbot ./cmd/squirrelbot

build-web: web/node_modules
	npm --prefix web run build

image:
	$(CONTAINER) build -t $(IMAGE) .

test:
	go test ./...

test-e2e: web/node_modules
	go test -tags e2e -count=1 ./e2e
	npm --prefix web exec playwright install chromium
	npm --prefix web run e2e

web/node_modules: web/package.json web/package-lock.json
	npm --prefix web ci
