# GVPN - Go VPN Server

A simple, secure VPN server written in Go with cross-platform support.

## Features

- 🔒 TLS encryption for secure connections
- 🔐 ChaCha20-Poly1305 AEAD encryption for data packets
- 👥 Client management and VPN IP assignment
- 💓 Keepalive mechanism for connection stability
- 🌐 Cross-platform TUN interface support (Linux, macOS, Windows)
- 🔑 User authentication with token-based sessions
- 📊 Real-time client monitoring
- 🛣️ IP routing configuration
- 📝 Comprehensive logging

## Requirements

- Go 1.21 or higher
- Linux/macOS/Windows
- Administrative/root privileges for TUN interface creation

## Installation

### Clone and build

```bash
git clone https://github.com/yingkitw/gvpn.git
cd gvpn
go mod download
go build -o bin/server ./cmd/server
go build -o bin/client ./cmd/client
```

### Using Make

```bash
make build
```

## Configuration

Create a configuration file at `~/.gvpn/config.json` or use the example:

```json
{
  "listen_addr": ":1194",
  "vpn_network": "10.8.0.0/24",
  "mtu": 1500,
  "debug": true
}
```

### Configuration Options

- `listen_addr`: Server listening address (default: `:1194`)
- `vpn_network`: VPN network in CIDR notation (default: `10.8.0.0/24`)
- `mtu`: Maximum Transmission Unit (default: `1500`)
- `debug`: Enable debug logging (default: `false`)

## Usage

### Starting the Server

```bash
# Using built binary
sudo ./bin/server

# Using Make
make run-server

# With custom configuration
sudo ./bin/server -config /path/to/config.json
```

### Starting the Client

```bash
# Using built binary
./bin/client

# Using Make
make run-client

# Connect to custom server
./bin/client -server 192.168.1.100:1194
```

## Default Credentials

The server comes with a default user:
- Username: `admin`
- Password: `admin123`

**Important:** Change the default credentials in production!

## Server Commands

Once the server is running, you can use these commands:

- `clients` - List connected clients
- `routes` - Show routing table
- `help` - Show available commands
- `quit` or `exit` - Shutdown the server

## Platform-Specific Notes

### Linux

TUN interface support is fully functional. Requires root privileges.

### macOS

Basic TUN support is available. You may need to install additional drivers:

```bash
brew install tuntaposix
```

### Windows

Limited TUN support. Consider using a third-party TAP driver for full functionality.

## Security Features

- **TLS Encryption**: All connections are encrypted using TLS 1.2+
- **AES/ChaCha20 Encryption**: Data packets use modern AEAD encryption
- **Token-based Authentication**: Secure session management
- **Automatic Key Rotation**: Encryption keys are generated per session
- **Keepalive Mechanism**: Detects and removes dead connections

## Architecture

```
gvpn/
├── cmd/
│   ├── client/          # VPN client implementation
│   └── server/          # VPN server implementation
├── internal/
│   ├── auth/            # Authentication and authorization
│   ├── config/          # Configuration management
│   ├── logger/          # Logging system
│   ├── routing/         # IP routing configuration
│   ├── server/          # Core server logic
│   └── tunnel/          # TUN interface handling
├── examples/            # Example configurations
└── Makefile            # Build automation
```

## Development

### Running Tests

```bash
make test
```

### Code Formatting

```bash
make fmt
```

### Linting

```bash
make lint
```

### Adding New Features

1. Follow the existing code structure
2. Add appropriate logging using the logger package
3. Update configuration if needed
4. Test on multiple platforms
5. Update documentation

## Troubleshooting

### TUN Interface Creation Fails

- Ensure you have root/administrative privileges
- Check if TUN/TAP drivers are installed
- Verify VPN network configuration

### Clients Cannot Connect

- Check firewall settings
- Verify server is running and listening
- Check network connectivity
- Review debug logs

### Performance Issues

- Adjust MTU size
- Check network bandwidth
- Monitor system resources
- Consider reducing debug logging

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is provided as-is for educational purposes.

## Disclaimer

This VPN implementation is intended for educational purposes and should not be used in production environments without proper security review and hardening.

## Support

For issues and questions:
- Open an issue on GitHub
- Check existing documentation
- Review debug logs

## Roadmap

- [ ] Web management interface
- [ ] Enhanced authentication methods
- [ ] Multi-server support
- [ ] Load balancing
- [ ] Advanced routing features
- [ ] Performance optimizations
- [ ] Comprehensive test suite
- [ ] Docker containerization