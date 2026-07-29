# DevPanel Project Specification (`devpanel.yaml`)

`devpanel.yaml` is the declarative infrastructure configuration for **DevPanel PaaS**. It defines how multi-tier applications (frontend, backend, databases, workers) in single repositories or monorepos are cloned, built, networked, and deployed.

Place this file in the **root directory** of your Git repository.

---

## Access & Hosting Architecture

### 1. Default Subpath Access (No Custom Domain Required)
Before configuring a custom domain, every application project deployed on DevPanel is **automatically hosted and accessible** at:

$$\text{http://140.245.116.79/app/<project-name>/}$$

*(e.g., `http://140.245.116.79/app/my-monorepo-startup/`)*

### 2. Custom Domain Access (Automatic HTTPS / SSL)
When you assign a domain in `devpanel.yaml` (e.g. `app.yourdomain.com`), DevPanel automatically provisions SSL certificates via Caddy and routes traffic directly to the container:

$$\text{https://app.yourdomain.com/}$$

---

## Complete Example (`devpanel.yaml`)

```yaml
version: "1.0"
project: "my-monorepo-startup"

services:
  # --------------------------------------------------
  # 1. Frontend Static App (SvelteKit, React, Vue, Vite)
  # --------------------------------------------------
  frontend:
    type: static
    source:
      repo: "https://github.com/username/my-monorepo.git"
      branch: "main"
      directory: "web" # Monorepo sub-directory
    build:
      engine: "node"
      command: "npm ci && npm run build"
      output_dir: "build"
    domains:
      - "app.yourdomain.com"

  # --------------------------------------------------
  # 2. Backend API Service (Go, Node.js, Python, Rust)
  # --------------------------------------------------
  backend:
    type: web
    source:
      repo: "https://github.com/username/my-monorepo.git"
      branch: "main"
      directory: "api"
    build:
      engine: "dockerfile"
      dockerfile_path: "Dockerfile"
      args:
        ENV: "production"
    deploy:
      port: 8080
      command: "./server"
    resources:
      cpu_limit: "1.0"
      mem_limit: "512m"
    env:
      - key: DB_HOST
        value: "database"
      - key: DB_PORT
        value: "5432"
      - key: DB_PASS
        secret: "postgres_password"

  # --------------------------------------------------
  # 3. Relational Database (PostgreSQL / MySQL)
  # --------------------------------------------------
  database:
    type: database
    image: "postgres:15-alpine"
    deploy:
      port: 5432
    volumes:
      - name: "pg_data"
        mount_path: "/var/lib/postgresql/data"
    env:
      - key: POSTGRES_DB
        value: "appdb"
      - key: POSTGRES_USER
        value: "admin"
      - key: POSTGRES_PASSWORD
        secret: "postgres_password"

  # --------------------------------------------------
  # 4. In-Memory Cache (Redis)
  # --------------------------------------------------
  redis-cache:
    type: database
    image: "redis:7-alpine"
    deploy:
      port: 6379
```

---

## Detailed Schema Reference

### 1. Root Attributes

| Field | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `version` | String | **Yes** | Schema specification version. Must be `"1.0"`. |
| `project` | String | **Yes** | Unique project identifier slug (e.g., `my-monorepo-startup`). Used for path URL routing (`/app/<project>`) and network isolation. |
| `services` | Map | **Yes** | Map of microservices to provision. The service key (e.g., `backend`, `database`) becomes the internal network hostname. |

---

### 2. Service Classification (`services.<name>.type`)

- **`static`**: Client-side single page app (Svelte, React, Vue) compiled into static HTML/CSS/JS served via automated Nginx.
- **`web`**: Long-running HTTP backend processes (Node.js, Go, Python, Rust) exposed through internal container ports.
- **`database`**: Pre-built container images pulled directly from Docker Hub without requiring a build step (PostgreSQL, Redis, MongoDB).
- **`worker`**: Background queue processors or cron jobs without public HTTP endpoints.

---

### 3. Source Configuration (`source`)

Configures source code retrieval for services.

| Field | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `repo` | String | **No** | Git repository URL (e.g., `https://github.com/user/another-repo.git`). **Defaults to current repository** when omitted, or set to `"current"`, `"this"`, or `"self"`. Only required when referencing a *different* external repository. |
| `branch` | String | No | Target Git branch, tag, or commit hash. Defaults to `main`. |
| `directory` | String | No | Sub-directory path inside the monorepo where the microservice code lives (e.g., `web`, `api`). Use `.` or omit for root directory. |

---

### 4. Build Engine Options (`build`)

| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `engine` | String | `dockerfile` | Build engine type: `dockerfile`, `node`, `go`, or `static`. |
| `dockerfile_path` | String | `Dockerfile` | Relative path to the `Dockerfile` inside `source.directory`. |
| `command` | String | None | Shell command executed during build phase (e.g., `npm ci && npm run build`). |
| `output_dir` | String | `build` / `dist` | Folder containing compiled static assets (required for `type: static`). |
| `args` | Map | `{}` | Build-time arguments passed to `docker build --build-arg`. |

---

### 5. Runtime & Network Deployment (`deploy`)

| Field | Type | Description |
| :--- | :--- | :--- |
| `port` | Integer | Internal listening TCP port inside the container (e.g., `8080`, `5432`, `3000`). |
| `command` | String | Overrides the default container `ENTRYPOINT` / `CMD`. |

---

### 6. Environment Variables & Secrets (`env`)

Environment variables support both static string values and secure secret references:

```yaml
env:
  # Plaintext value
  - key: PORT
    value: "8080"

  # Encrypted secret reference stored in DevPanel UI
  - key: JWT_SECRET
    secret: "production_jwt_secret"
```

---

### 7. Persistent Volumes (`volumes`)

Mounts persistent Docker storage to ensure database or upload data survives container restarts:

```yaml
volumes:
  - name: "pg_data"                        # Named Docker volume
    mount_path: "/var/lib/postgresql/data"  # Mount point inside container
```

---

### 8. Resource Limits (`resources`)

Hard CPU and Memory caps to prevent noisy neighbor containers:

```yaml
resources:
  cpu_limit: "0.5" # 0.5 CPU core limit
  mem_limit: "512m" # 512 Megabytes RAM limit
```

---

### 9. Custom Domain Routing (`domains`)

List of custom domain names routed automatically via DevPanel's Caddy reverse proxy:

```yaml
domains:
  - "api.mycompany.com"
  - "app.mycompany.com"
```

---

## 🤖 AI Prompting Guide (Generate `devpanel.yaml` for your Project)

Copy and paste the prompt below into any AI LLM (ChatGPT, Claude, Gemini) to generate a valid `devpanel.yaml` for your specific repository stack:

```text
Act as a Senior DevOps Engineer. Generate a valid `devpanel.yaml` configuration file for my application based on the following project architecture:

- Project Name: "my-app"
- Monorepo Subfolders:
  - Frontend: "web" (SvelteKit static app, listens on port 3000)
  - Backend: "api" (Go REST API with Dockerfile, listens on port 8080)
  - Database: PostgreSQL 15 Alpine (port 5432) with a persistent volume named "db_data"
- Git Repository URL: "https://github.com/username/my-monorepo.git"

Ensure the YAML strictly adheres to the DevPanel 1.0 schema specification:
1. Version must be "1.0"
2. Use type: static for frontend, type: web for backend, and type: database for postgres
3. Configure environment variables for DB_HOST="database", DB_PORT="5432", and DB_PASS using secret: "db_password"
4. Output valid, clean YAML without markdown wrapper explanations.
```