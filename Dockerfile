# --- build stage ---
FROM golang:1.25-alpine AS build
WORKDIR /src

# Cache deps first.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/radio ./cmd/server

# --- runtime stage ---
FROM alpine:3.21
# ffmpeg provides ffprobe, used to probe track durations during scanning.
RUN apk add --no-cache ffmpeg && adduser -D -u 10001 radio
WORKDIR /app
COPY --from=build /out/radio /app/radio

# Volumes mounted at runtime: music library (ro) and art cache (rw).
ENV RADIO_MUSIC_DIRS=/music \
    RADIO_ART_CACHE_DIR=/data/art \
    RADIO_LISTEN_ADDR=:8080
EXPOSE 8080
USER radio
ENTRYPOINT ["/app/radio"]
