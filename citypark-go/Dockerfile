FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/server /app/server
ENV APP_ENV=production UPLOAD_DIR=/data/uploads
RUN mkdir -p /data/uploads && chown nobody /data/uploads
VOLUME /data/uploads
EXPOSE 3000
USER nobody
ENTRYPOINT ["/app/server"]
