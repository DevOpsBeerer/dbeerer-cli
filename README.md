# 🍺 DevOpsBeerer CLI

**DevOpsBeerer** is a CLI tool for managing OIDC/OAuth2 playground environments. It helps you deploy infrastructure and manage beer-themed scenarios to learn and experiment with authentication flows.

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 🎯 What is DevOpsBeerer?

DevOpsBeerer is an educational playground for learning OIDC/OAuth2 concepts through hands-on scenarios. It provides:

- **🚀 Easy Infrastructure Setup** - Deploy K3s with Keycloak, cert-manager, and ingress controller
- **📋 Multiple Scenarios** - Pre-built scenarios for different authentication flows
- **🍺 Beer-Themed Fun** - Learn serious concepts through a fun beer management application
- **🔧 CLI Management** - Simple commands to manage everything

## 🚀 Quick Start

### Prerequisites

- **Go 1.21+** (for building from source)
- **Git** (for cloning repositories)
- **Bash** (for running setup scripts)
- **Linux/macOS** (K3s requirement)

### Installation

```bash
# Clone the CLI repository
git clone https://github.com/DevOpsBeerer/dbeerer-cli.git
cd dbeerer-cli

# Build the CLI
go mod tidy
go build -o dbeerer

# Move to PATH (optional)
sudo mv dbeerer /usr/local/bin/
```

### 🎬 Getting Started

```bash
# 1. Deploy infrastructure (K3s + Keycloak + cert-manager + ingress)
dbeerer infra deploy

# 2. List available scenarios
dbeerer list

# 3. Start a scenario
dbeerer start scenario-1

# 4. Check status
dbeerer status

# 5. Stop scenario when done
dbeerer stop
```

## 📚 Commands Reference

### Infrastructure Management

```bash
# Deploy complete infrastructure
dbeerer infra deploy

# Check infrastructure status
dbeerer infra status
```

### Scenario Management

```bash
# List all available scenarios
dbeerer list

# Start a specific scenario
dbeerer start <scenario-id>

# Stop current scenario
dbeerer stop

# Check overall status
dbeerer status
```

### Cleanup

```bash
# Remove everything
dbeerer cleanup

# Keep infrastructure, remove only scenarios
dbeerer cleanup --keep-infra
```

## 🏗️ Architecture

### Infrastructure Components

- **K3s Cluster** - Lightweight Kubernetes distribution
- **Keycloak** - Identity and Access Management (SSO)
- **Cert-Manager** - Automatic SSL certificate management
- **Ingress Controller** - Traffic routing and SSL termination

### Scenario Management

- Scenarios are fetched from [`DevOpsBeerer/playground-scenarios-charts`](https://github.com/DevOpsBeerer/playground-scenarios-charts)
- Each scenario is a Helm chart with complete applications
- Only one scenario runs at a time (automatic cleanup)

## 🔧 Configuration

### Default Settings

- **Namespace**: scenario ID
- **Domain**: `devopsbeerer.local`
- **Helm Release**: scenario ID

### Environment Variables

```bash
# Helm driver (optional)
export HELM_DRIVER=secret

# Kubernetes config (if not default)
export KUBECONFIG=/path/to/kubeconfig
```

## 🐛 Troubleshooting

### Common Issues

**Infrastructure deployment fails:**

```bash
# Check if scripts are accessible
ls -la ~/.../playground/install-k3s.sh
ls -la ~/.../playground/init-k3s.sh

# Check logs during deployment
dbeerer infra deploy 2>&1 | tee deploy.log
```

**Scenario won't start:**

```bash
# Check if infrastructure is running
dbeerer infra status

# Verify scenario exists
dbeerer list

# Check Kubernetes cluster
kubectl cluster-info
kubectl get pods -A
```

**Can't access applications:**

```bash
# Check ingress
kubectl get ingress -A

# Check services
kubectl get svc -A

# Verify /etc/hosts entries for *.playground.local
```

### Debug Commands

```bash
# Check Helm releases
helm list -A

# Check Kubernetes resources
kubectl get all -n devopsbeerer

# View logs
kubectl logs -n devopsbeerer -l app=your-app
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Project Structure

```text
dbeerer-cli/
├── cmd/                    # CLI commands
│   ├── root.go            # Root command
│   ├── infrastructure.go  # Infrastructure commands  
│   ├── scenario.go        # Scenario commands
│   ├── list.go           # List scenarios
│   └── other.go          # Status/cleanup
├── internal/              # Internal packages
│   ├── infrastructure/   # Infrastructure management
│   ├── scenarios/        # Scenario operations
│   ├── helm/            # Helm integration
│   └── github/          # GitHub API client
├── main.go              # Entry point
├── go.mod              # Go modules
└── README.md           # This file
```

## 📖 Learning Resources

### OIDC/OAuth2 Concepts

- **OpenID Connect** - Identity layer on top of OAuth 2.0
- **OAuth2 Flows** - Authorization Code, Client Credentials, etc.
- **JWT Tokens** - JSON Web Tokens for secure information transmission
- **PKCE** - Proof Key for Code Exchange for public clients

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙋 Support

- **Issues**: [GitHub Issues](https://github.com/DevOpsBeerer/dbeerer-cli/issues)
- **Discussions**: [GitHub Discussions](https://github.com/DevOpsBeerer/dbeerer-cli/discussions)
- **Documentation**: [DevOpsBeerer Docs](https://devopsbeerer.github.io)

---

**🍺 Happy Learning with DevOpsBeerer!**

*Remember: With great beer comes great responsibility... to understand authentication flows properly!*
