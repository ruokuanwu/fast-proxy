# fast-proxy

`fast-proxy` is a command-line tool for local development environments. It quickly maps local domains to services running on your machine.

A common use case is proxying `app.test` to `localhost:3000`. The tool automatically maintains `/etc/hosts` and Caddy configuration for you.

## Features

- Add local domain reverse proxy rules.
- Remove proxy rules by rule ID, with unique ID prefix matching.
- List proxy rules managed by the tool.
- Check the runtime environment and Caddy status.
- Resync hosts and Caddy configuration from the state file.
- Automatically maintain records marked with `# fast-proxy` in `/etc/hosts`.
- Automatically maintain Caddy site snippet files.
- Run `caddy reload` automatically after configuration changes.
- Support `FAST_PROXY_HOME` to customize the state file directory.

## How it works

When a local proxy rule is added, `fast-proxy` does the following:

1. Writes the rule to `~/.fast-proxy/config.json`.
2. Adds a domain mapping to `/etc/hosts`.
3. Generates a Caddy site configuration under `/etc/caddy/fast-proxy/`.
4. Reloads Caddy.

For example:

```text
app.test -> localhost:3000
```

Generates a Caddy configuration like this:

```caddyfile
app.test {
    tls internal
    reverse_proxy localhost:3000
}
```

## Requirements

- Go 1.22+
- Caddy installed, with the `caddy` command available in `PATH`
- Permission to modify `/etc/hosts`, `/etc/caddy/Caddyfile`, and `/etc/caddy/fast-proxy`

Because system files are modified, most write operations usually need to run with `sudo`.

If Caddy is not installed yet, run:

```bash
fp doctor
```

This command prints installation suggestions and the current environment check results.

## Installation

Build from source:

```bash
make build
```

Install to `/usr/local/bin/fast-proxy`:

```bash
make install
```

After installation, two commands are available:

- `fast-proxy`
- `fp`

Uninstall:

```bash
make uninstall
```

You can also run the CLI directly from source:

```bash
make run ARGS="list"
```

## Initialize Caddy configuration

Before first use, initialize the Caddy integration:

```bash
sudo fast-proxy init
```

Or use the short command:

```bash
sudo fp init
```

This command adds the fast-proxy import line to the system Caddyfile:

```caddyfile
import /etc/caddy/fast-proxy/*.caddy
```

It also creates the site snippet directory:

```text
/etc/caddy/fast-proxy
```

If Caddy is not detected, `init` prints installation instructions. You can also run `fp doctor` first to inspect the environment.

## Usage

### Add a proxy rule

```bash
sudo fast-proxy add <domain> <host:port>
```

Example:

```bash
sudo fast-proxy add app.test localhost:3000
```

Short command:

```bash
sudo fp add app.test localhost:3000
```

After the command runs:

- `/etc/hosts` contains a record managed by `fast-proxy`.
- `/etc/caddy/fast-proxy/app.test.caddy` is generated.
- Caddy is reloaded.

### List rules

```bash
sudo fast-proxy list
```

Short command:

```bash
fp list
```

Alias:

```bash
sudo fast-proxy ls
```

Example output:

```text
+--------------+----------+----------------+
| ID           | DOMAIN   | TARGET         |
+--------------+----------+----------------+
| a1b2c3d4e5f6 | app.test | localhost:3000 |
+--------------+----------+----------------+
```

### Remove rules

Remove a rule by its ID:

```bash
sudo fast-proxy remove <id>
```

Example:

```bash
sudo fast-proxy remove a1b2c3d4e5f6
```

Short command:

```bash
sudo fp rm a1b2c3d4e5f6
```

ID prefix matching is supported as long as the prefix is unique:

```bash
sudo fast-proxy remove a1b2c3
```

### Check the environment

```bash
fp doctor
```

This command checks whether:

- Caddy is installed.
- The Caddyfile exists.
- The fast-proxy import is configured.
- The site snippet directory exists.
- The state file is valid.
- The Caddy configuration passes validation.
- The Caddy service is running.

### Resync configuration

If `/etc/hosts` or `/etc/caddy/fast-proxy/*.caddy` was changed or deleted manually, resync from the state file:

```bash
sudo fp sync
```

You can also remove multiple rules at once:

```bash
sudo fast-proxy remove a1b2c3 d4e5f6
```

Remove command aliases:

```bash
sudo fast-proxy rm <id>
sudo fast-proxy delete <id>
```

## Commands

| Command | Description |
| --- | --- |
| `fast-proxy init` | Initialize the system Caddy configuration. |
| `fast-proxy doctor` | Check the runtime environment and Caddy status. |
| `fast-proxy sync` | Resync hosts and Caddy configuration from the state file. |
| `fast-proxy add <domain> <host:port>` | Add or update a proxy rule. |
| `fast-proxy list` | List current rules. |
| `fast-proxy remove <id> [id...]` | Remove rules. |

After installation, you can also use the short command `fp`, such as `fp init`, `fp add`, `fp list`, and `fp rm`.

## File locations

| Path | Description |
| --- | --- |
| `~/.fast-proxy/config.json` | fast-proxy state file. |
| `/etc/hosts` | System hosts file. |
| `/etc/caddy/Caddyfile` | System Caddyfile. |
| `/etc/caddy/fast-proxy/*.caddy` | Site configuration generated by fast-proxy. |

When running through `sudo`, if `SUDO_USER` exists, the state file is written to the original user's home directory instead of `/root`.

You can also customize the state file directory with `FAST_PROXY_HOME`:

```bash
FAST_PROXY_HOME=/path/to/home sudo fast-proxy list
```

## Validation

### domain

- Cannot be empty.
- Cannot be `localhost`.
- Cannot contain spaces, `/`, or `:`.

### target

- `localhost:port` or `127.0.0.1:port` is recommended.
- The port range is `1-65535`.
- A host or IP without a port is also accepted for maintaining hosts records.

## Notes

- `fast-proxy` only removes records in `/etc/hosts` that contain the `# fast-proxy` marker.
- `fast-proxy` regenerates site snippets under `/etc/caddy/fast-proxy/*.caddy`.
- Local targets only include `localhost` and `127.0.0.1`; these rules generate Caddy reverse proxy configuration.
- Non-local targets are written to hosts records, but no Caddy reverse proxy site snippets are generated.
- If Caddy reload fails, make sure Caddy is installed, the service is running, and `/etc/caddy/Caddyfile` is valid.
- `fast-proxy doctor` checks the current environment, Caddy installation status, and configuration issues.
- `fast-proxy sync` rebuilds hosts records and site snippets from the state file.

## Development

Common development commands:

```bash
make fmt
make test
make tidy
make build
```

Project structure:

```text
cmd/fast-proxy/main.go      CLI entry point
internal/app/app.go         Command definitions and core flow
internal/config/            State file and path configuration
internal/hosts/             Hosts file synchronization
internal/caddy/             Caddy configuration generation and reload
```