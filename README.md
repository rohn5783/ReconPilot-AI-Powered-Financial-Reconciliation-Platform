# ReconPilot: Enterprise External Attack Surface Management (EASM) & AI Security Reporting

ReconPilot is a hardened, production-grade External Attack Surface Management (EASM) platform, automated DAG reconnaissance engine, and AI-powered security synthesis workstation.

Built with **Go 1.23+**, **Gin**, **PostgreSQL**, **Redis**, the **Official Google GenAI SDK (`google.golang.org/genai`)**, and a bespoke industrial **React 19 + TypeScript + Vite** web interface designed with Swiss precision (Linear / Cloudflare Radar / Palantir Foundry aesthetic).

---

## Architecture Overview

```
                                      +-----------------------------+
                                      |   Target Scope Registry     |
                                      |   (DNS TXT Ownership Check) |
                                      +--------------+--------------+
                                                     |
                                                     v
                                      +-----------------------------+
                                      |     Zero-SSRF Guardrail     |
                                      |   (RFC 1918 / Metadata Ban) |
                                      +--------------+--------------+
                                                     |
                                                     v
                      +--------------------------------------------------------------+
                      |               Concurrent DAG Pipeline Engine                 |
                      |  [Discovery] -> [Live Prober] -> [Port Scan] -> [Auditor]    |
                      +------------------------------+-------------------------------+
                                                     |
                                                     v
                                      +-----------------------------+
                                      |  Historical Drift Engine    |
                                      |  (State Mutation Tracking)  |
                                      +--------------+--------------+
                                                     |
                                                     v
                                      +-----------------------------+
                                      |   Google GenAI Synthesis    |
                                      |   (Gemini 2.5 Flash / Pro)  |
                                      |   + Deterministic Fallback  |
                                      +--------------+--------------+
                                                     |
                                                     v
                      +--------------------------------------------------------------+
                      |                     Dual Delivery Interfaces                 |
                      |   1. Standalone CLI Binary (`reconpilot scan -t acme.org`)   |
                      |   2. High-Density Web Workstation (`React 19` + `Xterm.js`)  |
                      +--------------------------------------------------------------+
```

---

## Key Features

1. **Hardened Pipeline Runner (`pkg/runner`):**
   - Zero command injection: uses strictly tokenized `exec.CommandContext` without shell interpolation.
   - Bounded ring buffers prevent OOM when streaming voluminous tool outputs.
   - Subprocess-group termination (`syscall.Kill(-pgid, syscall.SIGKILL)`) on context cancellation or stage timeout.

2. **Scope Authorization & SSRF Blocker (`internal/security`):**
   - Strict CIDR and DNS TXT verification token checks (`_reconpilot-challenge.<domain>`).
   - Hard blocks on RFC 1918, loopback (`127.0.0.0/8`), link-local metadata (`169.254.169.254`), CGNAT, and IPv6 unique local addresses.

3. **Autonomous Tool Adapters (`internal/modules`):**
   - `discovery`: Subdomain enumeration with streaming JSON parser.
   - `prober`: Live HTTP/HTTPS status, page title, and technology stack fingerprinting.
   - `portscan`: TCP port and service banner discovery.
   - `auditor`: Vulnerability detection with CVSS scoring, CVE mapping, and raw request/response capture.
   - `dry_run / mock_mode`: Realistic synthetic attack surface simulation for instant offline verification.

4. **Historical Asset Drift Engine (`internal/service`):**
   - Automatically tracks state diffs against baseline scans: newly discovered hosts, decommissioned subdomains, newly exposed ports, and resolved vs. new findings.

5. **Official Google GenAI SDK (`internal/ai`):**
   - Integrated with `google.golang.org/genai`.
   - Token optimization: deduplicates findings before prompting Gemini.
   - Produces structured JSON output containing executive posture briefs, overall risk score ($0–100$), exploitability rankings, and defensive hardening playbooks.
   - Includes a deterministic rule-based fallback engine for resilient offline operations.

6. **Swiss Industrial Web Workstation (`frontend/`):**
   - Carbon & Obsidian palette (`#08090C`, `#0D0F14`, `#13161F`), hairline `#1E222D` borders, `0px`–`4px` micro-radii, tabular-nums monospace font.
   - Data Grids: `@tanstack/react-table` + `@tanstack/react-virtual` for high-density asset exploration.
   - Interactive Topology: `@xyflow/react` + `dagre` DAG visualization.
   - Authentic Live Console: `@xterm/xterm` with ANSI streaming over WebSocket / SSE.
   - Global Command Palette: `cmdk` accessible via `Cmd+K` / `Ctrl+K`.

---

## Quickstart Guide

### Option 1: Standalone CLI (Zero Dependencies)

The standalone CLI binary runs immediately with built-in in-memory state tracking:

```bash
# Display CLI commands
./reconpilot-cli.exe --help

# Execute a dry-run reconnaissance pipeline with Markdown export
./reconpilot-cli.exe scan -t acme.org --dry-run --export markdown

# Export structured JSON report
./reconpilot-cli.exe scan -t acme.org --dry-run --export json
```

### Option 2: Full Web Workstation & API Server

Start the backend API server (which automatically serves the compiled React 19 frontend SPA from `frontend/dist`):

```bash
# Run the API server
./reconpilot-api.exe
```

Access the UI at:
- Web Workstation: **`http://localhost:8080`**
- Health Probe: **`http://localhost:8080/healthz`**
- Prometheus Metrics: **`http://localhost:8080/metrics`**

### Option 3: Docker Compose (PostgreSQL 16 + Redis 7 + Backend + Nginx Frontend)

To spin up the multi-container production stack:

```bash
# Set your Gemini API key (optional, falls back to deterministic engine if unset)
export GEMINI_API_KEY="your-gemini-api-key"

# Build and start services
docker-compose up -d --build
```

Access services:
- Web Interface: **`http://localhost:5173`**
- Backend API: **`http://localhost:8080`**
- PostgreSQL: **`localhost:5432`** (`reconpilot` / `reconpilot_secret`)
- Redis: **`localhost:6379`**

---

## Repository Structure

```text
reconpilot/
├── backend/
│   ├── cmd/
│   │   ├── api/                 # HTTP/WebSocket API server entrypoint
│   │   └── cli/                 # Standalone CLI binary entrypoint
│   ├── internal/
│   │   ├── config/              # Typed configuration with viper & validation
│   │   ├── domain/              # Scope, Asset, ScanJob, Finding, DriftDiff, AIReport
│   │   ├── security/            # Scope guardrails, SSRF blocker, DNS TXT challenge
│   │   ├── orchestrator/        # Concurrent DAG engine, bounded worker pools, checkpoints
│   │   ├── modules/             # Pluggable tool adapters (discovery, prober, portscan, auditor)
│   │   ├── ai/                  # Google GenAI Go SDK client & fallback synthesis engine
│   │   ├── repository/          # PostgreSQL (pgx/v5) & Redis (go-redis/v9) persistence
│   │   ├── service/             # Historical drift diffing engine & report exporter
│   │   └── transport/
│   │       ├── http/            # Gin handlers, auth middleware, CORS, Prometheus
│   │       └── stream/          # WebSocket & SSE real-time telemetry streaming
│   ├── pkg/
│   │   ├── runner/              # Hardened os/exec runner with ring buffers
│   │   └── validator/           # RFC 1123 FQDN, CIDR, and URL validators
│   └── migrations/              # 001_init.sql relational schema
├── frontend/
│   ├── src/
│   │   ├── api/                 # Strongly typed API client with high-fidelity mock fallbacks
│   │   ├── components/
│   │   │   ├── command/         # Cmd+K industrial command palette
│   │   │   ├── dag/             # DAG execution pipeline stepper
│   │   │   ├── layout/          # TopHeader, Sidebar, BottomTelemetryDock, WorkspaceLayout
│   │   │   ├── modals/          # LaunchScanModal
│   │   │   ├── terminal/        # Xterm.js streaming ANSI terminal console
│   │   │   └── views/           # AssetInventoryView, TopologyView, AIAnalystView, ScopeRegistryView, EvidenceDrawer
│   │   ├── store/               # Zustand workspace state
│   │   └── types/               # TypeScript domain interfaces
│   ├── package.json             # React 19, Vite, TanStack Table/Query, React Flow, Xterm.js
│   └── vite.config.ts
├── docker/
│   └── nginx.conf               # Hardened reverse-proxy configuration
├── Dockerfile                   # Multi-stage production container build
├── docker-compose.yml           # Complete orchestration setup
└── README.md
```

---

## License & Security Compliance

This platform is engineered strictly for authorized security assessments and perimeter defense posture management. Active scanning without authorization is strictly prohibited.

