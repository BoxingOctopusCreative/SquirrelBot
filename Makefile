IMAGE ?= squirrelbot:0.1.0

.DEFAULT_GOAL := help

.PHONY: help dev bot web db up down build build-bot build-web image test

help:
	@printf '%s\n' \
		'dev          Postgres, Air, and the dashboard' \
		'bot          Air only' \
		'web          Dashboard dev server' \
		'db           Postgres in Docker' \
		'up           Bot and Postgres in Docker' \
		'down         Stop the Docker services' \
		'build        Local bot binary and dashboard' \
		'image        Container image ($(IMAGE))' \
		'test         Go tests'

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
	docker compose up postgres -d

up:
	docker compose up --build

down:
	docker compose down

build: build-bot build-web

build-bot:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/squirrelbot ./cmd/squirrelbot

build-web: web/node_modules
	npm --prefix web run build

image:
	docker build -t $(IMAGE) .

test:
	go test ./...

web/node_modules: web/package.json web/package-lock.json
	npm --prefix web ci
