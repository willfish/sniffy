# Sniffy - AWS Secrets Manager Analysis Tool

A beautiful, interactive CLI tool for analyzing and managing potentially unused AWS Secrets Manager secrets. Built with the Charm TUI libraries for a delightful terminal experience.

## ✨ Features

- **🔍 Smart Secret Analysis** - Automatically identifies secrets that haven't been accessed in 14+ days
- **🎨 Beautiful Interface** - Tokyo Night theme with smooth animations and intuitive navigation
- **📊 Interactive Table** - Browse secrets with keyboard navigation and multi-select capabilities
- **🔒 Version Management** - View and reveal different versions of secrets
- **⚡ Fuzzy Filtering** - Quick search through secrets with include/exclude patterns
- **📋 Clipboard Integration** - Copy secret names with a single keystroke
- **🗑️ Safe Deletion** - Multi-select and confirm deletion of unused secrets
- **🚀 Real-time Scanning** - Live progress indicators during AWS operations

## 🚀 Quick Start

### Prerequisites

- AWS credentials configured (via `aws configure`, environment variables, or IAM roles)
- AWS Secrets Manager read/write permissions

### Installation

#### Option 1: Install from Release (Recommended)

**One-line install script:**
```bash
curl -fsSL https://raw.githubusercontent.com/willfish/sniffy/main/install | bash
```

**Manual download:**
```bash
# Download for your platform from releases
# Linux AMD64
curl -L -o sniffy https://github.com/willfish/sniffy/releases/latest/download/sniffy-linux-amd64
chmod +x sniffy
sudo mv sniffy /usr/local/bin/

# Linux ARM64
curl -L -o sniffy https://github.com/willfish/sniffy/releases/latest/download/sniffy-linux-arm64
chmod +x sniffy
sudo mv sniffy /usr/local/bin/

# macOS AMD64 (Intel)
curl -L -o sniffy https://github.com/willfish/sniffy/releases/latest/download/sniffy-darwin-amd64
chmod +x sniffy
sudo mv sniffy /usr/local/bin/

# macOS ARM64 (Apple Silicon)
curl -L -o sniffy https://github.com/willfish/sniffy/releases/latest/download/sniffy-darwin-arm64
chmod +x sniffy
sudo mv sniffy /usr/local/bin/
```

#### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/willfish/sniffy
cd sniffy

# Install dependencies
go mod tidy

# Build and install
go build -o sniffy
sudo mv sniffy /usr/local/bin/
```

#### Option 3: Go Install

```bash
go install github.com/willfish/sniffy@latest
```

### Verify Installation

```bash
sniffy --version
# or just run
sniffy
```

### AWS Configuration

Ensure your AWS credentials are configured with the necessary permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "secretsmanager:ListSecrets",
                "secretsmanager:DescribeSecret",
                "secretsmanager:GetSecretValue",
                "secretsmanager:ListSecretVersionIds",
                "secretsmanager:DeleteSecret"
            ],
            "Resource": "*"
        }
    ]
}
```

## 🎮 Usage

### Main Interface

After launching, Sniffy Scan will automatically connect to AWS and scan for potentially unused secrets (not accessed in 14+ days).

### Navigation

#### Results View
- **↑/↓ or j/k** - Navigate through secrets
- **Space** - Select/deselect secrets for deletion
- **Enter** - View secret versions and details
- **y** - Copy secret name to clipboard
- **/** - Filter secrets (include matching)
- **?** - Filter secrets (exclude matching)
- **Shift+D** - Delete selected secrets (with confirmation)
- **r** - Rescan for unused secrets only
- **R** - Rescan all secrets
- **esc** - Clear current filter
- **q** - Quit application

#### Secret Details View
- **↑/↓** - Navigate through versions
- **r** - Reveal secret value for selected version
- **y** - Copy secret name to clipboard
- **esc** - Return to main results
- **q** - Quit application

#### Filter Mode
- **Type** - Enter search terms for fuzzy matching
- **Enter** - Apply filter
- **esc** - Cancel filter

### Filtering Examples

```bash
# Include secrets containing "prod"
/ → prod → Enter

# Exclude secrets containing "test"
? → test → Enter

# Clear any active filter
esc
```

## 🔧 Configuration

### Scan Threshold

By default, secrets not accessed in 14+ days are considered "potentially unused". You can modify this in the code:

```go
const recentThresholdDays = 14  // Change this value
```

### Theme Customization

The Tokyo Night theme colors can be customized in the color definitions:

```go
var (
    bgColor     = lipgloss.Color("#1a1b26")
    fgColor     = lipgloss.Color("#a9b1d6")
    blueColor   = lipgloss.Color("#7aa2f7")
    purpleColor = lipgloss.Color("#bb9af7")
    greenColor  = lipgloss.Color("#9ece6a")
    redColor    = lipgloss.Color("#f7768e")
    yellowColor = lipgloss.Color("#e0af68")
    dimColor    = lipgloss.Color("#565f89")
)
```

## 📋 Features in Detail

### Secret Analysis
- Fetches all secrets from AWS Secrets Manager
- Filters out configuration secrets (ending in "-configuration")
- Calculates days since last access
- Identifies potentially unused secrets based on configurable threshold

### Interactive Selection
- Multi-select interface with checkboxes
- Visual feedback for selected items
- Bulk operations on selected secrets

### Version Management
- View all versions of a secret
- See creation dates, stages, and access history
- Reveal secret values on demand

### Safe Deletion
- Confirmation prompts before deletion
- Clear error reporting if deletions fail
- Automatic refresh after successful deletions

## 🛡️ Security Considerations

- **Read-only by default** - Scanning operations don't modify anything
- **Explicit confirmation** - Deletion requires explicit user confirmation
- **No credential storage** - Uses standard AWS credential chain
- **Minimal permissions** - Only requests necessary AWS permissions
- **Secure clipboard** - Secret values are only copied when explicitly requested

## 🐛 Troubleshooting

### Common Issues

**"Failed to initialize AWS connection"**
- Ensure AWS credentials are configured
- Check your AWS region settings
- Verify IAM permissions for Secrets Manager

**"No secrets found"**
- Check if you have secrets in the current AWS region
- Verify your AWS account has access to Secrets Manager
- Ensure your credentials have list permissions

**Cursor positioning issues**
- This is a known issue with the table component
- The functionality works correctly despite visual cursor jumping

### Debug Mode

For detailed error information, you can add debug output by modifying the error handling sections in the code.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Creating Releases

Releases are automated via GitHub Actions. To create a new release:

1. Create and push a new tag:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. The GitHub Action will automatically:
   - Build binaries for all supported platforms
   - Create a GitHub release
   - Upload all binaries as release assets

### Supported Platforms

| OS      | Architecture | Binary Name                  |
|---------|-------------|------------------------------|
| Linux   | AMD64       | `sniffy-linux-amd64`        |
| Linux   | ARM64       | `sniffy-linux-arm64`        |
| macOS   | AMD64       | `sniffy-darwin-amd64`       |
| macOS   | ARM64       | `sniffy-darwin-arm64`       |

### Development Setup

```bash
# Clone the repository
git clone https://github.com/willfish/sniffy
cd sniffy

# Install dependencies
go mod tidy

# Run in development mode
go run main.go

# Build for current platform
go build -o sniffy

# Build for all supported platforms (requires goreleaser or manual builds)
make build-all
```

### Release Builds

The project uses automated releases that build for multiple platforms:

- **Linux**: AMD64, ARM64
- **macOS**: AMD64 (Intel), ARM64 (Apple Silicon)  

Binaries are automatically built and uploaded to GitHub Releases using the naming convention:
`sniffy-{os}-{arch}`

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Charm](https://charm.sh/) TUI libraries
- Inspired by the need for better AWS secrets management
- Tokyo Night theme colors

## 📞 Support

If you encounter issues or have questions:

1. Check the [Issues](https://github.com/willfish/sniffy/issues) page
2. Review the troubleshooting section above
3. Create a new issue with detailed information about your problem

### Install Script Issues

If the install script fails:
```bash
# Check if curl is installed
curl --version

# Manually specify architecture if detection fails
ARCH=amd64 bash <(curl -fsSL https://raw.githubusercontent.com/willfish/sniffy/main/install.sh)

# Or download manually from releases page
# https://github.com/willfish/sniffy/releases/latest
```

---

**Made with ❤️ and 🐕 by the Sniffy Scan team**
