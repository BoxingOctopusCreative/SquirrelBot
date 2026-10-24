FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/squirrelbot ./cmd/squirrelbot

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/squirrelbot /squirrelbot
COPY --from=web /src/web/build /web/build
ENV WEB_DIR=/web/build
ENV HTTP_ADDR=:8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/squirrelbot"]
