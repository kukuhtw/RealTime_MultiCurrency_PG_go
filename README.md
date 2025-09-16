# Payment Gateway POC (Go, gRPC, Docker Compose)

Monorepo **payment-gateway-poc** berisi beberapa layanan Go (HTTP & gRPC) yang disusun untuk percobaan arsitektur payment gateway: **api-gateway**, **payments**, **fx**, **wallet**, **risk**, beserta observability (**Prometheus** & **Grafana**).

> Bahasa: Indonesia · Toolchain: Go 1.22 (default), bisa upgrade ke 1.23 jika mau.

---

## 1) Ringkasan

* **api-gateway** – HTTP entrypoint (port `8080`) yang memanggil layanan-layanan di belakangnya.
* **services/** – layanan HTTP kecil per domain (`payments`, `fx`, `wallet`, `risk`).
* **cmd/\*-grpc** – server gRPC per domain (port `9091..9094`), dengan metrik Prometheus (port `9101..9104`).
* **proto/gen/** – paket kode hasil generate dari file `.proto` (import path: `github.com/example/payment-gateway-poc/proto/gen/...`).
* **Observability** – Prometheus (port `9090`) + Grafana (port `3000`) dengan provisioning dashboard.

> **Catatan penting**: Kita **tidak** lagi tergantung repo eksternal `github.com/kukuhtw/RealTime_MultiCurrency_PG_go`. Semua import diarahkan ke path modul **lokal** `github.com/example/payment-gateway-poc/...`. Pastikan file hasil generate (`*.pb.go`) **ada** di `proto/gen/**`.

---

## 2) Struktur Repo (ringkas)

```
.
├── cmd/
│   ├── payments-grpc/
│   ├── fx-grpc/
│   ├── wallet-grpc/
│   └── risk-grpc/
├── services/
│   ├── api-gateway/
│   ├── payments/
│   ├── fx/
│   ├── wallet/
│   └── risk/
├── proto/
│   └── gen/
│       ├── common/v1/*.proto + *.pb.go
│       ├── fx/v1/*.proto + *.pb.go
│       ├── wallet/v1/*.proto + *.pb.go
│       ├── risk/v1/*.proto + *.pb.go
│       └── payments/v1/*.proto + *.pb.go
├── deployments/
│   ├── compose/docker-compose.dev.yaml
│   └── docker/Dockerfile
├── grafana/
│   ├── grafana_payment_gateway_dashboard.json
│   └── provisioning/**
├── prometheus/prometheus.yml
└── go.mod, go.sum, Makefile, README.md
```

---

## 3) Prasyarat

* **Docker** & **Docker Compose** terinstal.
* Tidak perlu Go di host (semua build & tooling berjalan di container).

(Optional) Jika ingin generate protobuf di host, butuh `protoc` dan plugin `protoc-gen-go` & `protoc-gen-go-grpc`.

---

## 4) Quick Start

```bash
# dari root repo
make dev
# atau setara dengan:
docker compose -f deployments/compose/docker-compose.dev.yaml up -d --build
```

Lalu akses:

* API Gateway: [http://localhost:8080](http://localhost:8080)
* Prometheus:  [http://localhost:9090](http://localhost:9090)
* Grafana:     [http://localhost:3000](http://localhost:3000)  (user: admin / pass: admin)

> **Jika build gagal** lihat bagian **Troubleshooting** di bawah.

---

## 5) Layanan & Port

**HTTP services**

* api-gateway → `8080:8080`
* payments → `8081:8081`
* fx → `8082:8082`
* wallet → `8083:8083`
* risk → `8084:8084`

**gRPC services**

* payments-grpc → `9091` (metrics `9101`)
* fx-grpc → `9092` (metrics `9102`)
* wallet-grpc → `9093` (metrics `9103`)
* risk-grpc → `9094` (metrics `9104`)

> Ketika semua container jalan di Compose, **jangan** dial ke `localhost`. Gunakan **nama service** Compose, mis.: `risk-grpc:9094`, `wallet-grpc:9093`, `fx-grpc:9092` dari service **payments**.

---

## 6) Docker Compose (dev)

File: `deployments/compose/docker-compose.dev.yaml`

* Setiap service build memakai **ARG `SERVICE`** untuk memilih target direktori build.

  * Contoh: `services/api-gateway` (HTTP) atau `cmd/payments-grpc` (gRPC).
* `depends_on` sudah diset seperlunya.
* api-gateway me-mount static assets (read-only) dari `services/api-gateway/static`.
* Prometheus & Grafana diprovision dengan file di repo.

Jalankan:

```bash
docker compose -f deployments/compose/docker-compose.dev.yaml up -d --build
# stop & hapus kontainer:
docker compose -f deployments/compose/docker-compose.dev.yaml down
```

---

## 7) Dockerfile (multi-stage)

File: `deployments/docker/Dockerfile`

Pola umum:

```dockerfile
# syntax=docker/dockerfile:1.6
FROM golang:1.22-alpine AS build
WORKDIR /app

# cache modul
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# source
COPY . .

# (opsional tapi dianjurkan)
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod tidy && go mod download && go mod verify

ARG SERVICE=services/api-gateway
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    sh -c "cd ${SERVICE} && go build -trimpath -ldflags='-s -w' -o /out/app"

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/app /app/app
# aman untuk service lain kalau folder ada; hapus jika tidak perlu
COPY services/api-gateway/static /app/static
EXPOSE 8080
ENTRYPOINT ["/app/app"]
```

**Tips penting**:

* Pakai `sh -c` (bukan `sh -lc`) → mencegah PATH environment "hilang" saat build.
* Jangan `rm -f go.sum && go mod tidy` sembarangan di Dockerfile; cukup `go mod tidy`.
* Pastikan `.dockerignore` **tidak** meng-ignore `proto/**` atau `*.pb.go`.

---

## 8) Protobuf & gRPC

Semua import code gRPC diarahkan ke path **lokal**:

```
github.com/example/payment-gateway-poc/proto/gen/<svc>/v1
```

### 8.1 Sumber schema

* Letakkan file `.proto` di struktur berikut:

```
proto/gen/common/v1/*.proto
proto/gen/fx/v1/*.proto
proto/gen/wallet/v1/*.proto
proto/gen/risk/v1/*.proto
proto/gen/payments/v1/*.proto
```

* Set **go\_package** di setiap `.proto` agar sesuai modul:

```proto
option go_package = "github.com/example/payment-gateway-poc/proto/gen/<svc>/v1;<pkgname>";
```

Contoh: `<svc>=payments`, `<pkgname>=paymentsv1`.

### 8.2 Generate `*.pb.go`

> **Pilih salah satu opsi** sesuai versi Go base image.

**Opsi A – Go 1.22 (pin versi plugin)**

```bash
docker run --rm -v "$PWD":/work -w /work golang:1.22-alpine sh -c '
  set -e
  apk add --no-cache protobuf git
  go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32.0
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0
  export PATH="$PATH:/go/bin"

  PROTO_COUNT=$(find proto/gen -name "*.proto" | wc -l || true)
  if [ "$PROTO_COUNT" = "0" ]; then
    echo "ERROR: tidak ada file .proto di proto/gen/**"; exit 1; fi

  protoc -I . \
    --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    $(find proto/gen -name "*.proto" | sort)
'
```

**Opsi B – Go 1.23 (boleh `@latest`)**

1. Ubah semua `golang:1.22-alpine` → `golang:1.23-alpine` (Dockerfile & perintah tooling).
2. Lalu generate:

```bash
docker run --rm -v "$PWD":/work -w /work golang:1.23-alpine sh -c '
  set -e
  apk add --no-cache protobuf git
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
  export PATH="$PATH:/go/bin"
  protoc -I . \
    --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    $(find proto/gen -name "*.proto" | sort)
'
```

**Verifikasi**

```bash
find proto -maxdepth 4 -type f -name '*pb.go' | sort
```

---

## 9) Manajemen Dependensi Go (go.mod / go.sum)

Semua perintah dilakukan **di dalam container**:

```bash
docker run --rm -v "$PWD":/work -w /work golang:1.22-alpine \
  sh -c 'apk add --no-cache git && go mod tidy && go mod download && go mod verify'
```

> Penting: `git` wajib ada supaya `go mod` dapat mengambil modul.

Jika muncul error **missing go.sum entry** (xxhash, protobuf, x/sys, dst), jalankan lagi perintah di atas setelah memastikan internet OK.

---

## 10) Observability

* **Prometheus** (port `9090`) pakai konfigurasi dari `prometheus/prometheus.yml`.
* **Grafana** (port `3000`) sudah diprovision:

  * Credentials default: `admin` / `admin`.
  * Dashboard JSON dimount ke `/var/lib/grafana/dashboards/payment.json`.
  * Seluruh provisioning dimount dari `grafana/provisioning/`.

---

## 11) Troubleshooting (umum)

### a) `yaml: line XX: did not find expected key`

* Periksa indentasi di `docker-compose.dev.yaml` (spasi, bukan tab). Pastikan setiap service sejajar.

### b) `sh: go: not found`

* Terjadi jika step build dijalankan pada **runtime image** (alpine) bukan pada **build image** (`golang:*`). Pastikan compile terjadi di stage `FROM golang:... AS build`.
* Gunakan `sh -c` (bukan `-lc`).

### c) `no required module provides package github.com/example/payment-gateway-poc/proto/gen/...`

* Artinya `*.pb.go` **belum ada** → generate dari `.proto` (lihat bagian **8.2**), atau **copy** dari repo lama.
* Jika file `.proto` juga belum ada, copy dulu schema-nya ke `proto/gen/**`.

### d) `missing go.sum entry for module ...`

* Jalankan `go mod tidy && go mod download && go mod verify` di container (butuh `apk add git`).

### e) `fatal: could not read Username for 'https://github.com'`

* Go mencoba fetch modul `github.com/example/payment-gateway-poc/proto/gen/...` dari GitHub karena paket lokal tidak ditemukan.
* Solusi: pastikan folder `proto/gen/**` berisi `*.pb.go` (atau `.proto` + generate). Setelah itu `go mod tidy` lagi.

### f) `.dockerignore` menghapus file yang dibutuhkan

* Pastikan **tidak** ada pola yang mengabaikan `proto/**` atau `*.pb.go`.

---

## 12) Perintah Berguna

```bash
# rebuild total tanpa cache
docker compose -f deployments/compose/docker-compose.dev.yaml build --no-cache

# lihat log salah satu service
docker compose -f deployments/compose/docker-compose.dev.yaml logs -f payments

# cek daftar file pb.go
find proto -maxdepth 4 -type f -name '*pb.go' | sort

# hapus kontainer + network (tidak menghapus volume)
docker compose -f deployments/compose/docker-compose.dev.yaml down
```

---

## 13) FAQ

**Q: Kenapa tidak pakai repo `RealTime_MultiCurrency_PG_go`?**
A: Untuk menghindari dependency eksternal & masalah versi, semua import diarahkan ke paket **lokal** `github.com/example/payment-gateway-poc/proto/gen/...`. Artinya schema & hasil generate berada di repo ini sendiri.

**Q: Harus upgrade ke Go 1.23?**
A: Tidak wajib. Jika tetap di Go 1.22, **pin** plugin `protoc-gen-go` ke `v1.32.0` dan `protoc-gen-go-grpc` ke `v1.3.0` (lihat bagian **8.2 Opsi A**). Jika upgrade ke Go 1.23, kamu bisa gunakan `@latest`.

---

## 14) Lisensi

Tentukan lisensi yang sesuai (MIT/BSD/Apache-2.0) sesuai kebutuhan proyek.

---

### Catatan Akhir

* Pastikan import path di kode konsisten dengan `module` di `go.mod`: `module github.com/example/payment-gateway-poc`.
* Selalu verifikasi `proto/gen/**` berisi `*.pb.go` sebelum build.
* Jika menambah service baru, cukup tambahkan target di Compose dengan `args.SERVICE` menunjuk ke direktori yang benar (HTTP: `services/<svc>`, gRPC: `cmd/<svc>-grpc`).
