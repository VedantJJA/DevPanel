DevPanel Deployment Guide (Oracle Cloud OCI)This guide provides a step-by-step approach to deploying the DevPanel application on an Oracle Cloud Infrastructure (OCI) Ubuntu VM using Systemd Socket Activation for "Scale-to-Zero" capability.1. Oracle Cloud Infrastructure (OCI) SetupBefore deploying, you must configure the OCI network to allow web traffic.Security Lists: Navigate to your VCN > Public Subnet > Default Security List.Add Ingress Rules:Source Type: CIDRSource CIDR: 0.0.0.0/0IP Protocol: TCPDestination Port: 80, 443Note: Ping (ICMP) is blocked by OCI default rules. Use curl to test your connection, not ping.2. Server Environment SetupSSH into your Ubuntu VM and prepare the environment.# Update system
sudo apt update && sudo apt upgrade -y

# Install Go, Node.js, and git
sudo apt install -y golang nodejs npm git

# Create your project directory
mkdir -p ~/devpanel
cd ~/devpanel
git clone <YOUR_REPO_URL> .
3. Building the ApplicationYou must build the Svelte UI and the Go binary.# Build the UI
cd ui
npm install
npm run build
cd ..

# Build the Go binary
go build -o devpanel ./cmd/server/main.go
4. Systemd Socket ActivationThis is the "Scale-to-Zero" logic. It allows the OS to start your app only when a request hits port 80/443.Create the Socketsudo vim /etc/systemd/system/devpanel.socket[Unit]
Description=DevPanel Socket Activation

[Socket]
ListenStream=80
ListenStream=443

[Install]
WantedBy=sockets.target
Create the Servicesudo vim /etc/systemd/system/devpanel.service[Unit]
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
5. Deployment & PermissionsApply the configuration and move the binary to the execution path.# Move the binary
sudo cp ~/devpanel/devpanel /usr/local/bin/devpanel
sudo chmod +x /usr/local/bin/devpanel

# Fix Permissions for the SQLite database
# SQLite needs to write .wal and .shm files in the project folder
sudo chown -R ubuntu:ubuntu /home/ubuntu/devpanel
sudo chmod -R 775 /home/ubuntu/devpanel

# Start the service
sudo systemctl daemon-reload
sudo systemctl enable devpanel.socket
sudo systemctl start devpanel.socket
sudo systemctl start devpanel.service
6. TroubleshootingIf the service fails to start:Check Status: sudo systemctl status devpanel.serviceView Logs: sudo journalctl -u devpanel.service -n 50 --no-pagerCommon Issue (Database Path): If you see unable to open database file, ensure the WorkingDirectory in your .service file is correct and that the ubuntu user owns the folder.Connection Timeout: If curl times out, check if iptables is blocking the port:sudo iptables -I INPUT 5 -m state --state NEW -p tcp --dport 80 -j ACCEPT
sudo iptables -I INPUT 5 -m state --state NEW -p tcp --dport 443 -j ACCEPT
7. Deployment ScriptCreate ~/deploy.sh for easy updates:#!/bin/bash
cd ~/devpanel
git pull origin main
cd ui && npm run build && cd ..
go build -o devpanel ./cmd/server/main.go
sudo systemctl stop devpanel.service
sudo cp devpanel /usr/local/bin/
sudo systemctl start devpanel.service
echo "Deployment Complete."
