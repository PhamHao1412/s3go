# S3Go 🚀

S3Go is a secure, local, high-performance Amazon S3 web client built in Go and wrapped in a premium, glassmorphic single-page application (SPA). Designed with a **100% CGO-free architecture**, S3Go is cross-platform, light on resources, and extremely easy to compile and deploy on macOS, Windows, Linux, or cloud environments like Render.

---

## ✨ Features

- **🎨 Sleek Glassmorphic Interface:** A modern, visual-first dark mode dashboard utilizing subtle backdrop blur filters (`backdrop-filter`), harmonious HSL colors, premium typography (Outfit and Inter from Google Fonts), and fluid micro-animations.
- **⚡ Massive Bucket Listing Optimization (Root Bypass):** 
  - Effortlessly browse large S3 buckets containing millions of objects or deep directory structures.
  - By bypassing the root folder listing and targeting a nested directory directly, S3Go loads the dashboard instantly (**0.0-second listing time**), avoiding AWS timeout issues (`context deadline exceeded`).
  - Configurable dynamically via environment variables (`BYPASS_BUCKET` and `BYPASS_FOLDER`).
- **📤 Multi-Threaded Direct Uploads with Progress Bars:**
  - Files are streamed directly from the browser to Amazon S3 using **S3 Presigned PUT URLs** via `XMLHttpRequest`.
  - Bypasses backend network bottlenecks completely while displaying real-time, highly accurate percentage progress bars.
- **🔒 Secure Connection Manager & Editor:**
  - Add, list, edit, or delete S3 connection profiles directly from the client.
  - **AES-256-GCM Encryption:** Secret access keys are securely encrypted at rest in the database.
  - **Edit Isolation:** AWS Secret Access Keys are never returned to the frontend. During edits, the secret field remains blank. Leaving it blank retains the existing secure key, making credential updates easy and safe.
  - **Pre-flight Validation:** Automatically executes a pre-flight S3 verification check using the decrypted credentials before saving the connection.
- **🗃️ Dual Storage Backend (JSON & PostgreSQL):**
  - **CGO-Free Local JSON Fallback:** Uses a lightweight, thread-safe JSON file database (`connections.json`) wrapped in double-checked read/write mutexes (`sync.RWMutex`) for zero-configuration local development.
- **⌨️ Keyboard Shortcuts & History Navigation:**
  - **Escape Key Closing:** Close any active modal (Connection Manager, Folder Creation, File Preview, or Delete confirmation) instantly by pressing the `Esc` key.
  - **Browser Back/Forward Support:** Uses the HTML5 History API to synchronize current connection and prefix folders as query parameters (`?connection=ID&prefix=PATH`). You can navigate folder levels natively using browser back/forward buttons, and page reloads (`F5`) automatically restore your exact browsing state.

---

## 📂 Project Architecture

```
s3go/
├── cmd/
│   └── serverd/
│       ├── main.go           # Bootstrapper, database initializer, and HTTP server daemon
│       ├── router/           # Routing configuration mapping HTTP paths to controllers
│       └── static/           # Embedded Frontend SPA Assets (index.html, style.css, app.js)
├── internal/
│   ├── app/
│   │   └── configs.go        # Configuration loader, environment validation & .env parser
│   ├── entity/
│   │   └── connection.go     # Core database entity definition
│   ├── model/
│   │   ├── connection.go     # Connection API contracts and schemas
│   │   ├── s3.go             # S3 file model structures & presigned payloads
│   │   └── types.go          # General model types
│   ├── persistence/
│   │   ├── spec.go           # Unit of Work standard interfaces & implementation
│   │   └── connection/       # Connection repository specs & local/PG integrations
│   ├── infra/
│   │   └── s3/               # AWS SDK client wrapper client & credentials validation
│   ├── service/
│   │   ├── connection/       # Connection manager business service
│   │   └── s3/               # S3 file action business service
│   ├── controller/
│   │   └── rest/v1/          # REST API controller handlers and response DTO mapping
│   └── pkg/
│       └── crypto/           # AES-256-GCM symmetric encryption & hermetic unit tests
├── .env.example              # Template configuration for environment settings
├── Makefile                  # Build, test, run, and cleanup commands
└── go.mod                    # CGO-free dependencies list
```

## 🚀 Getting Started

### 📋 Prerequisites
- Go 1.22+ installed locally.
- Git.

### 🛠️ Development & Commands

S3Go includes a simple `Makefile` to automate all repetitive actions:

- **Build S3Go binary:**
  ```bash
  make build
  ```
- **Start the S3Go server:**
  ```bash
  make run
  ```
  Once started, the application will boot and serve the client interface. Open your browser and navigate to:
  [http://localhost:8080](http://localhost:8080)

- **Run hermetic unit tests:**
  ```bash
  make test
  ```

- **Clean build artifacts:**
  ```bash
  make clean
  ```

## 🛡️ License

This project is open-source. Feel free to use, modify, and distribute it in compliance with standard software guidelines.
