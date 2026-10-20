FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/server ./cmd/server
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/userctl ./cmd/userctl
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/feature-worker ./cmd/feature-worker

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app && apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/server /app/server
COPY --from=build /out/userctl /app/userctl
COPY --from=build /out/feature-worker /app/feature-worker
COPY --from=build /src/configs /app/configs
COPY --from=build /src/migrations /app/migrations
USER app
EXPOSE 8080
ENTRYPOINT ["/app/server"]
