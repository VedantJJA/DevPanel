# DevPanel

DevPanel is a self-hosted developer platform for managing containerized applications with dynamic routing.

## Environment Configuration

Configure the following environment variables in `.env`:

```env
ROOT_DOMAIN=example.com
ROUTING_MODE=path   # or "subdomain"
```

### DNS Requirements for Subdomain Mode
To use `ROUTING_MODE=subdomain`:
1. Create an **A** (or **CNAME**) record for `panel.example.com` pointing to the panel host IP.
2. Create a wildcard **A** (or **CNAME**) record for `*.example.com` pointing to the panel host IP.

### Reverse Proxy & Caddy Configuration
When using Caddy or Nginx in front of DevPanel:
- In both path mode and subdomain mode, route `panel.{$ROOT_DOMAIN}` and `{$ROOT_DOMAIN}` to the DevPanel entrypoint.
- In subdomain mode, route `*.{$ROOT_DOMAIN}` to the DevPanel entrypoint with wildcard TLS certificates.
