# DevPanel

DevPanel is a lightweight, self-hosted Cloud Application Platform for hosting full-stack applications with Go, Docker, SvelteKit, and Caddy.

## Environment Configuration

Copy `.env.example` to `.env` or set environment variables:

```text
ROOT_DOMAIN=example.com
ROUTING_MODE=path   # or "subdomain"
```

- `ROOT_DOMAIN`: The base domain for the panel and hosted sites.
- `ROUTING_MODE`:
  - `path`: Hosted projects are mounted at `<rootDomain>/app/<project-name>`.
  - `subdomain`: Hosted projects are served on `<project-name>.<rootDomain>`.

## DNS Requirements for Subdomain Mode

To use `ROUTING_MODE=subdomain`:
1. Create an **A** (or **CNAME**) record for `panel.example.com` pointing to the panel host IP.
2. Create a wildcard **A** (or **CNAME**) record for `*.example.com` pointing to the panel host IP.

## Reverse Proxy & Caddy Configuration

When using Caddy or Nginx in front of DevPanel:
- In both path mode and subdomain mode, route `panel.{$ROOT_DOMAIN}` and `{$ROOT_DOMAIN}` to the DevPanel entrypoint.
- In subdomain mode, route `*.{$ROOT_DOMAIN}` to the DevPanel entrypoint with wildcard TLS certificates.
