# portwho

A small macOS command-line tool built with Go that shows which
process is listening on a TCP port and optionally stops it.

## Requirements

- macOS
- Go installed to build from source
- `lsof` available on your PATH

## Build

```sh
git clone https://github.com/zihvvn/portwho.git
cd portwho
go build -o portwho .
```

## Usage

Find processes listening on a port:

```sh
./portwho 3000
```

Example output:

```text
Port 3000 is used by node (PID 12345).
```

Request termination:

```sh
./portwho 3000 --kill
```

The command asks for confirmation before sending SIGTERM.
Only `y` or `yes` proceeds. It then checks whether a TCP listener
remains on the port.

## Limitations

- Supports TCP listeners only.
- Process visibility and termination depend on your permissions.
- Processes may ignore SIGTERM or restart automatically.
- Stopping a process affects everything it handles, including other ports.
