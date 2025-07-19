[![Build Status](https://github.com/AhmedAlYousif/moksarab/actions/workflows/build.yaml/badge.svg)](https://github.com/AhmedAlYousif/moksarab/actions/workflows/build.yaml)
[![Last Release](https://img.shields.io/github/v/release/AhmedAlYousif/moksarab?label=release)](https://github.com/AhmedAlYousif/moksarab/releases)

# moksarab

**moksarab** is a flexible, workspace-oriented API mocking server written in Go. It allows you to organize, create, and manage mock API responses, with optional workspace isolation, and is designed for easy deployment using pre-built binaries for all major platforms.

## Getting Started

### Downloading the Executable

1. Go to the [Releases page](https://github.com/AhmedAlYousif/moksarab/releases) on GitHub.
2. Download the binary for your operating system and architecture:
   - `moksarab-linux-x86_64`
   - `moksarab-linux-arm64`
   - `moksarab-windows-x86_64.exe`
   - `moksarab-windows-arm64.exe`
   - `moksarab-macos-x86_64`
   - `moksarab-macos-arm64`
3. (Linux/macOS) Make the binary executable:
   ```sh
   chmod +x moksarab-<your-os>-<your-arch>
   ```
4. (Optional) Rename or move the binary to a directory in your PATH for easier use.

> **Note:**  
> - `x86_64` = 64-bit Intel/AMD CPUs  
> - `arm64` = 64-bit ARM CPUs (Apple Silicon, Windows ARM laptops, some Linux devices)  
> - `macos` = macOS (Apple desktop/laptop)  
> - `windows` = Microsoft Windows  
> - `linux` = Linux distributions

### Configuration

Set environment variables as needed before running the executable:

- `PORT`: The port the server listens on (default: `8080`)
- `WORKSPACE_ENABLED`: Set to `true` to enable workspace support (default: `false`)
- `SQLITE_DB_PATH`: Path to the SQLite database file (default is in-memory if not set)

Example (Linux/macOS):
```sh
export PORT=8080
export WORKSPACE_ENABLED=true
export SQLITE_DB_PATH=./moksarab.db
./moksarab-linux-x86_64
```

Example (Windows, PowerShell, x86_64 or arm64):
```powershell
$env:PORT="8080"
$env:WORKSPACE_ENABLED="true"
$env:SQLITE_DB_PATH="./moksarab.db"
.\moksarab-windows-x86_64.exe
# or for ARM64
.\moksarab-windows-arm64.exe
```

### Running

Simply run the downloaded binary after setting your environment variables. The server will start on the configured port (default: 8080).

## API Overview

### Workspaces (if enabled)
- `POST /workspaces` — Create a new workspace
- `GET /workspaces` — List workspaces (supports `page` and `size` query params)
- `POST /workspaces/:workspaceId/mocks` — Create a new mock in a workspace
- `GET /workspaces/:workspaceId/mocks` — List mocks in a workspace

### Mocks (if workspaces are disabled)
- `POST /mocks` — Create a new mock
- `GET /mocks` — List mocks

### Mock Responses
- `POST /workspaces/:workspaceId/mocks/:mockId` — Add a response to a mock (workspace mode)
- `POST /mocks/:mockId` — Add a response to a mock (no workspace mode)

## Usage Examples

### Example 1: Workspace Enabled

Start the server with workspace support:
```sh
export WORKSPACE_ENABLED=true
export SQLITE_DB_PATH=./moksarab.db
./moksarab-linux-x86_64
```

1. Create a workspace:
   ```sh
   curl -X POST -H "Content-Type: application/json" -d '{"name":"test-ws"}' http://localhost:8080/workspaces
   ```
2. Create a mock in the workspace (replace `<workspaceId>` with the actual ID):
   ```sh
   curl -X POST -H "Content-Type: application/json" -d '{"path":"/hello","method":"GET","status":200,"response_body":"\"hello from sarab\""}' http://localhost:8080/workspaces/<workspaceId>/mocks
   ```
3. Call the mock route (replace `<workspaceId>` with the actual ID):
   ```sh
   curl http://localhost:8080/sarab/<workspaceId>/hello
   ```

### Example 2: Workspace Disabled

Start the server without workspace support:
```sh
export WORKSPACE_ENABLED=false
export SQLITE_DB_PATH=./moksarab.db
./moksarab-linux-x86_64
```

1. Create a mock:
   ```sh
   curl -X POST -H "Content-Type: application/json" -d '{"path":"/hello","method":"GET","status":200,"response_body":"\"hello from sarab\""}' http://localhost:8080/mocks
   ```
2. Call the mock route:
   ```sh
   curl http://localhost:8080/sarab/hello
   ```

## Developer Setup (Optional)

If you want to build or test from source:

1. Install Go 1.24.5 or newer.
2. Clone the repository:
   ```sh
   git clone https://github.com/AhmedAlYousif/moksarab
   cd moksarab
   ```
3. Install dependencies:
   ```sh
   go mod download
   ```
4. Run tests:
   ```sh
   go test ./...
   ```
5. Run from source:
   ```sh
   go run moksarab.go
   ```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
