<div align="center">

# 🪣 S3Go (`s3go`)

**Lightweight, CGO-free Amazon S3 web client and file manager built with Go.**

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![AWS SDK](https://img.shields.io/badge/AWS%20SDK-v2-FF9900?style=flat-square&logo=amazon-aws)](https://aws.amazon.com/sdk-for-go/)
[![Security](https://img.shields.io/badge/Encryption-AES--256--GCM-green?style=flat-square)](https://en.wikipedia.org/wiki/Galois/Counter_Mode)
[![Architecture](https://img.shields.io/badge/CGO--Free-Pure%20Go-007acc?style=flat-square)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)

</div>

---

S3Go is a self-contained, high-performance Amazon S3 web manager with an embedded dark-mode web interface. Built in 100% pure Go without CGO dependencies, it provides direct browser-to-S3 presigned uploads, root-bypass listing optimizations for massive buckets, AES-256-GCM credential encryption at rest, and dual-backend persistence (local JSON or PostgreSQL).

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│             Embedded Frontend SPA (Browser)                 │
│  - Glassmorphic UI / Dark mode                              │
│  - Direct-to-S3 Presigned Uploads with Progress Tracking    │
│  - HTML5 History Navigation (?connection=ID&prefix=PATH)   │
└─────────────────────────────┬───────────────────────────────┘
                              │
                              │ 1. API Requests (JSON)
                              v
┌─────────────────────────────────────────────────────────────┐
│                      S3Go Backend (Go)                      │
│                                                             │
│  +------------------------+     +------------------------+  │
│  |  Connection Controller |     |  S3 Action Controller  |  │
│  +------------------------+     +------------------------+  │
│              │                              │               │
│              v                              v               │
│  +------------------------+     +------------------------+  │
│  |  AES-256-GCM Crypto    |     |  AWS SDK v2 Client     |  │
│  |  (Key Encryption)      |     |  (Presign / List / DL) |  │
│  +------------------------+     +------------------------+  │
└──────────────┬──────────────────────────────┬───────────────┘
               │                              │
               │ 2. Credentials               │ 3. Generate Presigned URL
               v                              v
┌─────────────────────────────┐ ┌─────────────────────────────┐
│  Persistence Storage        │ │         Amazon S3           │
│  - Local JSON Fallback      │ │  - Direct Browser Uploads   │
│  - PostgreSQL / Supabase    │ │  - Bucket / Folder Listing  │
└─────────────────────────────┘ └─────────────────────────────┘
```

---

## Core Features

- **Direct Browser-to-S3 Uploads**: Files stream directly from client to S3 using Presigned PUT URLs, eliminating server-side bandwidth bottlenecks and providing accurate percentage upload progress.
- **Root-Bypass Bucket Listing Optimization**: Instantly open deeply nested folders in massive S3 buckets containing millions of objects without experiencing AWS listing timeouts (`BYPASS_BUCKET` & `BYPASS_FOLDER`).
- **Encrypted Credential Storage**: AWS Secret Access Keys are encrypted at rest using AES-256-GCM. Decrypted secrets are never exposed back to the client interface.
- **Dual Persistence Backends**:
  - **Zero-Config JSON**: Thread-safe file storage (`connections.json`) for local development.
  - **PostgreSQL / Supabase**: Database persistence for production or serverless environments.
- **100% CGO-Free**: Pure Go code that compiles into a single, static binary with zero external runtime dependencies.
- **Modern Embedded UI**: Self-contained single-page application served directly from the Go binary with keyboard shortcuts (`Esc` to dismiss modals) and URL state synchronization.

---

## Project Structure

```
.
├── cmd/
│   └── serverd/
│       ├── main.go           # Application bootstrap and HTTP daemon
│       ├── router/           # Route registration and static asset serving
│       └── static/           # Embedded frontend SPA assets (HTML, CSS, JS)
├── internal/
│   ├── app/                  # Environment loader and configuration models
│   ├── controller/rest/v1/   # REST API controllers and DTO mappings
│   ├── entity/               # Core domain models
│   ├── infra/s3/             # AWS SDK v2 client wrapper and verification
│   ├── persistence/          # Repository layer (JSON file & PostgreSQL)
│   ├── pkg/crypto/           # AES-256-GCM symmetric encryption utilities
│   └── service/              # Connection management and S3 file operations
├── .env.example              # Configuration template
├── Makefile                  # Build, test, and run automation
├── supabase_schema.sql       # PostgreSQL / Supabase schema definitions
└── go.mod                    # Dependencies
```

---

## Getting Started

### Prerequisites

- **Go**: `1.22` or higher

### 1. Configuration

Copy `.env.example` to `.env`:

```bash
cp .env.example .env
```

| Variable | Description | Default / Example |
|---|---|---|
| `PORT` | HTTP server port | `8080` |
| `S3GO_ENCRYPTION_KEY` | 32-character AES-256 secret key | `your-32-character-encryption-key` |
| `DB_PATH` | Path to JSON connections file | `connections.json` |
| `DATABASE_URL` | PostgreSQL connection string (optional) | - |
| `ALLOWED_IPS` | Comma-separated whitelist of allowed client IPs (optional) | - |
| `BYPASS_BUCKET` | S3 bucket name to enable root bypass optimization (optional) | - |
| `BYPASS_FOLDER` | Folder prefix to load directly on root bypass (optional) | - |

### 2. Build & Run

```bash
# Run locally
make run

# Or build binary
make build
./s3go
```

Open [http://localhost:8080](http://localhost:8080) in your browser to access the S3Go interface.

---

## Makefile Reference

| Command | Description |
|---|---|
| `make run` | Build and run S3Go locally |
| `make build` | Compile the static S3Go binary |
| `make test` | Run hermetic unit tests |
| `make clean` | Clean up build artifacts and caches |

---

## Security Model

- **Key Derivation & Cipher**: AES-256-GCM authenticated encryption ensures confidentiality and integrity of stored AWS credentials.
- **Pre-flight Validation**: Verifies S3 connection permissions before saving credentials.
- **Write-Only Secrets**: Edit actions in the UI do not disclose existing secret keys over the wire.

---

## License

This project is licensed under the [MIT License](LICENSE).
