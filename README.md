<div align="center">
  <img src="logo.png" alt="Rogue Logo" width="200" />
  <h1>Rogue</h1>
  <p>
    <strong>High-Performance HTTP/HTTPS Proxy</strong>
  </p>
  <p>
    <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go" alt="Go Version" /></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License" /></a>
  </p>
</div>

<br />

Rogue is a versatile, high-performance HTTP/HTTPS proxy server designed for deep traffic inspection and modification. Built with Go, it features automatic certificate generation for Man-in-the-Middle (MITM) capabilities, comprehensive request/response logging, and a flexible configuration system. It is ideal for debugging, security testing, and traffic analysis.

## Features

- **HTTP/HTTPS Proxy**: Full support for intercepting and inspecting HTTP and HTTPS traffic.
- **MITM Capabilities**: Automatically generates self-signed certificates to decrypt and inspect HTTPS traffic.
- **Session Logging**: Logs all requests and responses to JSON files for detailed analysis.
- **Session Export**: Export session logs to readable Markdown format for easy sharing and reporting.
- **Configurable**: Customize behavior via CLI flags or a `config.json` file.
- **Traffic Control**: (Coming Soon) Support for request/response modification and replay.

## Installation

### Prerequisites

- Go 1.21 or higher

### Install from Source

```bash
git clone https://github.com/standrze/rogue.git
cd rogue
go install .
```

Or run directly:

```bash
go run . start
```

## Usage

### Starting the Proxy

To start the proxy server with default settings (port 8080):

```bash
rogue start
```

**Options:**

- `--port` / `-p`: Port to listen on (default: 8080).
- `--host`: Host to bind to (default: "127.0.0.1").
- `--max-body-size`: Maximum size of request/response body to log in bytes (default: 1MB).

**Example:**

```bash
rogue start --port 9090 --max-body-size 524288
```

### Exporting Sessions

Rogue logs sessions to the `logs/` directory (or your configured session directory) by default. You can export these JSON logs to a formatted Markdown file.

```bash
rogue export <session_name> [output_path]
```

**Example:**

```bash
rogue export session_20251119_002232.json report.md
```

If `output_path` is omitted, it defaults to `<session_name>.md`.

## Configuration

Rogue looks for a `config.json` file in the current directory. You can use this to persist your configuration.

**Example `config.json`:**

```json
{
  "proxy": {
    "port": 8080,
    "host": "0.0.0.0",
    "timeout": 30
  },
  "certificate": {
    "auto_generate": true,
    "organization": "Rogue Proxy",
    "common_name": "Rogue CA",
    "valid_days": 365,
    "cert_path": "certs/ca.crt",
    "key_path": "certs/ca.key"
  },
  "logging": {
    "session_dir": "logs",
    "log_requests": true,
    "log_responses": true,
    "log_headers": true,
    "log_body": true,
    "max_body_size": 1048576
  }
}
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
