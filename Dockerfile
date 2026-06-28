# syntax=docker/dockerfile:1

# ---- build stage ----
# Go 1.26.4 で静的バイナリをビルドする。
FROM golang:1.26.4 AS build
WORKDIR /src

# 依存だけ先に取得してレイヤキャッシュを効かせる。
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO 無効の静的バイナリにして distroless/static で動かせるようにする。
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -trimpath -ldflags="-s -w" -o /out/anneal ./cmd/anneal

# ---- runtime stage ----
# 非 root・最小構成の distroless で実行する。
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/anneal /app/anneal

# Cloud Run は PORT を注入する。config がこれを尊重して待受アドレスを決める。
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/anneal"]
CMD ["serve"]
