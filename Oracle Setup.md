# DevPanel Deployment & Subdomain Setup Guide (Oracle Cloud OCI)

This guide provides step-by-step instructions for deploying DevPanel on an Oracle Cloud Infrastructure (OCI) Ubuntu VM, including **Subdomain-Based Routing**, Wildcard DNS configuration, and Systemd Socket Activation.

---

## 1. Oracle Cloud (OCI) Network & Firewall Setup

Before deploying, open ports **80**, **443**, and **8090** in your OCI VCN Security List and OS iptables firewall.

### A. OCI VCN Security List Rules
1. Navigate to: **OCI Console > Networking > Virtual Cloud Networks > Your VCN > Public Subnet > Default Security List**.
2. Add Ingress Rules:
   - **Source CIDR**: `0.0.0.0/0`
   - **IP Protocol**: `TCP`
   - **Destination Ports**: `80, 443, 8090`

### B. Ubuntu OS Firewall (iptables)
Run on your Oracle Ubuntu VM:
```bash
sudo iptables -I INPUT 5 -m state --state NEW -p tcp --dport 80 -j ACCEPT
sudo iptables -I INPUT 5 -m state --state NEW -p tcp --dport 443 -j ACCEPT
sudo iptables -I INPUT 5 -m state --state NEW -p tcp --dport 8090 -j ACCEPT
sudo netfilter-persistent save
```

---

## 2. Subdomain & Wildcard DNS Configuration

DevPanel automatically supports subdomain-based routing (`<project>.yourdomain.com` or `<service>.<project>.yourdomain.com`).

### Option A: Free Wildcard DNS via `nip.io` or `sslip.io` (Zero DNS Config Needed)
No domain purchase required! Append `.nip.io` to your Oracle Server Public IP:
- If your Oracle VM IP is `140.245.116.79`:
  - **DevPanel Dashboard**: `http://140.245.116.79:8090`
  - **Project App Subdomain**: `http://vtopcc.140.245.116.79.nip.io:8090`
  - **Backend Service Subdomain**: `http://vtopcc-backend.140.245.116.79.nip.io:8090`

### Option B: Custom Domain with Wildcard DNS
If you own a domain (e.g. `example.com`), add the following DNS records in Cloudflare / Namecheap / Route53:
1. **Root A Record**:
   - **Host**: `@`
   - **Type**: `A`
   - **Value**: `<YOUR_ORACLE_VM_IP>`
2. **Wildcard A Record** (Crucial for subdomains):
   - **Host**: `*`
   - **Type**: `A`
   - **Value**: `<YOUR_ORACLE_VM_IP>`

After adding `* -> <YOUR_ORACLE_IP>`, requests to `http://vtopcc.example.com` or `http://vtopcc-backend.example.com` will route directly into DevPanel!

---

## 3. Server Environment Setup

SSH into your Oracle Ubuntu VM:
```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Go, Node.js, Docker, and git
sudo apt install -y golang nodejs npm git docker.io
sudo usermod -aG docker ubuntu

# Create project directory
mkdir -p ~/devpanel
cd ~/devpanel
git clone <YOUR_REPO_URL> .
```

---

## 4. Building the Application

Build the SvelteKit frontend and Go binary:
```bash
# Build UI
cd ui
npm install
npm run build
cd ..

# Build Go binary
go build -o devpanel ./cmd/server/main.go
```

---

## 5. Systemd Service & Socket Activation Setup

Systemd Socket Activation enables "Scale-to-Zero" and automatic process restart.

### A. Create Socket: `/etc/systemd/system/devpanel.socket`
```ini
[Unit]
Description=DevPanel Socket Activation

[Socket]
ListenStream=80
ListenStream=443
ListenStream=8090

[Install]
WantedBy=sockets.target
```

### B. Create Service: `/etc/systemd/system/devpanel.service`
```ini
[Unit]
Description=DevPanel Service
Requires=devpanel.socket
After=network.target docker.service

[Service]
ExecStart=/usr/local/bin/devpanel
WorkingDirectory=/home/ubuntu/devpanel
User=ubuntu
Group=ubuntu
Restart=on-failure
RestartSec=5
NonBlocking=true

[Install]
WantedBy=multi-user.target
```

### C. Enable and Start
```bash
sudo cp ~/devpanel/devpanel /usr/local/bin/devpanel
sudo chmod +x /usr/local/bin/devpanel
sudo chown -R ubuntu:ubuntu /home/ubuntu/devpanel

sudo systemctl daemon-reload
sudo systemctl enable devpanel.socket
sudo systemctl start devpanel.socket
sudo systemctl start devpanel.service
```

---

## 6. Verification & Troubleshooting

- **Check Service Status**: `sudo systemctl status devpanel.service`
- **View Live Logs**: `sudo journalctl -u devpanel.service -n 50 -f`
- **Test Subdomain Routing**:
  ```bash
  curl -I -H "Host: vtopcc.140.245.116.79.nip.io" http://localhost:8090/
  ```
