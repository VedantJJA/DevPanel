# DevPanel Backend Implementation Todo List

This document lists all UI features, settings, and controls currently rendered in the frontend that require backend Go API endpoints, Docker integration, or background task runners to become fully operational.

---

## 🐘 1. PostgreSQL Database Services (`postgres`)

- [ ] **DB Credentials & Password Rotation**
  - Endpoint: `POST /api/projects/:id/services/:svc/reset-password`
  - Action: Update database password in container env, trigger SQL `ALTER USER postgres WITH PASSWORD ...` inside running container.
- [ ] **Storage Volume Expansion**
  - Endpoint: `POST /api/projects/:id/services/:svc/storage/resize`
  - Action: Dynamically resize Docker volume or update bind mount storage quotas.
- [ ] **Connection Pooling & SSL Certificates**
  - Endpoint: `GET /api/projects/:id/services/:svc/ssl-cert`
  - Action: Generate and download client SSL certificate authority bundle for encrypted database connections.
- [ ] **Database Backup & Restore Snapshots**
  - Endpoint: `POST /api/projects/:id/services/:svc/backups` & `POST /api/projects/:id/services/:svc/restore`
  - Action: Execute `pg_dump` to create `.sql.gz` snapshot, upload to storage/local path, and restore from backup file.

---

## 🌐 2. Static Sites & CDN Edge (`static`)

- [ ] **Domains & SSL Certificate Provisioning**
  - Endpoint: `POST /api/projects/:id/domains` & `DELETE /api/projects/:id/domains/:domain`
  - Action: Bind custom domain to container reverse proxy and issue automatic Let's Encrypt / ZeroSSL TLS certificates via ACME challenge.
- [ ] **URL Redirects & Rewrites**
  - Endpoint: `POST /api/projects/:id/redirects`
  - Action: Persist 301/302 redirects and SPA fallback rewrite rules (`/* -> /index.html`) to reverse proxy config.
- [ ] **Custom HTTP Headers**
  - Endpoint: `POST /api/projects/:id/headers`
  - Action: Configure CORS headers, Content-Security-Policy (CSP), and Strict-Transport-Security (HSTS) rules.
- [ ] **Edge Cache Invalidation & Purge**
  - Endpoint: `POST /api/projects/:id/cdn/purge`
  - Action: Flush static asset cache on proxy layer.

---

## 🚀 3. Web Services & Microservices (`web`)

- [ ] **Autoscaling & Replica Compute Sizing**
  - Endpoint: `POST /api/projects/:id/services/:svc/scale`
  - Action: Spin up additional container replicas (`docker service scale` or multiple container instances) and adjust CPU (`--cpus`) / Memory (`--memory`) limits.
- [ ] **Container Interactive Web Shell (Terminal)**
  - Endpoint: WebSocket `GET /ws/projects/:id/services/:svc/shell` or `POST /api/projects/:id/services/:svc/exec`
  - Action: Establish bi-directional PTY WebSocket session into container running `/bin/sh` or `/bin/bash` (`docker exec -it ...`).
- [ ] **PR Previews (Pull Request Environments)**
  - Endpoint: `POST /api/projects/:id/previews`
  - Action: Listen to GitHub PR webhooks, dynamically spin up isolated ephemeral containers for open PRs, and destroy them on PR close.
- [ ] **One-Off Maintenance & Migration Jobs**
  - Endpoint: `POST /api/projects/:id/jobs/run`
  - Action: Execute short-lived Docker containers to run database migration scripts (e.g. `npx prisma db push`, `rails db:migrate`).

---

## 🐳 4. Container Management & Host Operations (`/containers`)

- [ ] **Container JSON Inspection Modal**
  - Endpoint: `GET /api/containers/:id/inspect`
  - Action: Execute `docker inspect <container_id>` and return full raw JSON structure to UI modal.
- [ ] **Container Lifecycle Controls (Restart / Stop / Kill)**
  - Endpoint: `POST /api/containers/:id/restart`, `POST /api/containers/:id/stop`, `POST /api/containers/:id/kill`
  - Action: Send lifecycle signals via Docker API to stop or restart target containers.
- [ ] **Prune System Resources**
  - Endpoint: `POST /api/system/prune`
  - Action: Execute `docker system prune -a --volumes` to reclaim unused disk space.

---

## 🔒 5. Reverse Proxy & Traffic Routing (`/proxy`)

- [ ] **Dynamic Reverse Proxy Route Configuration**
  - Endpoint: `GET /api/proxy/routes` & `POST /api/proxy/routes`
  - Action: Manage incoming HTTP/HTTPS host header routing rules targeting specific container ports.
- [ ] **TLS / SSL Certificate Manager**
  - Endpoint: `GET /api/proxy/certs` & `POST /api/proxy/certs/upload`
  - Action: Upload custom SSL certificates (`.crt` / `.key`) or inspect automated Let's Encrypt renewals.
- [ ] **Upstream Proxy Health Check Status**
  - Endpoint: `GET /api/proxy/health`
  - Action: Probe proxy upstreams and return real-time latency and status codes.

---

## 💾 6. Storage & Persistent Volumes (`/volumes`)

- [ ] **Create & Provision Volume**
  - Endpoint: `POST /api/volumes`
  - Action: Create named Docker volume (`docker volume create <name>`) with specified driver options.
- [ ] **Attach Volume to Service**
  - Endpoint: `POST /api/volumes/:id/attach`
  - Action: Update service configuration to mount volume to target container path (e.g., `/var/lib/mysql`).
- [ ] **Volume Data Backup & Export**
  - Endpoint: `POST /api/volumes/:id/backup`
  - Action: Archive volume directory into `.tar.gz` for download or S3 upload.

---

## 📊 7. Monitoring, Historical Metrics & Alerts (`/metrics`)

- [ ] **Historical Time-Series Metrics Data**
  - Endpoint: `GET /api/metrics/historical?range=1h|24h|7d`
  - Action: Store CPU, Memory, Disk I/O, and Network stats in SQLite / time-series store and return time-series dataset.
- [ ] **Alert Threshold Triggers & Webhooks**
  - Endpoint: `POST /api/metrics/alerts`
  - Action: Set CPU/RAM usage threshold alerts and trigger Slack/Webhook notifications when exceeded.

---

## 🛠️ 8. Settings & GitHub Webhook Integration (`/settings`)

- [ ] **GitHub Webhook Automatic Setup**
  - Endpoint: `POST /api/github/webhooks/setup`
  - Action: Call GitHub API `POST /repos/{owner}/{repo}/hooks` to register push and PR deployment webhooks automatically.
- [ ] **Host Machine SSH Key Management**
  - Endpoint: `GET /api/settings/keys` & `POST /api/settings/keys`
  - Action: Generate and manage server SSH public keys for git cloning private repositories over SSH.
- [ ] **DevPanel Agent Auto-Updater**
  - Endpoint: `POST /api/system/update`
  - Action: Pull latest DevPanel release binary/container and perform zero-downtime hot reload.
