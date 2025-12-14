# blockblox

CLI tool for managing Roblox screen time limits.

**This is vibe coded software. No warranty. No guarantee. Use at your own risk.**

## Requirements

- macOS (Chrome cookie extraction uses macOS Keychain)
- Chrome browser with active Roblox login

## Installation

### Homebrew

```
brew tap astrostl/blockblox https://github.com/astrostl/blockblox
brew install blockblox
```

### Go

```
go install github.com/astrostl/blockblox@latest
```

Note: `blockblox --version` will show "dev" since go install doesn't include build-time version injection.

## Usage

```
# First time: extract credentials from Chrome
./blockblox init

# Get current screen time limit
./blockblox get

# Set screen time limit
./blockblox set 90        # 90 minutes
./blockblox set 90m       # 90 minutes
./blockblox set 4h        # 4 hours
./blockblox set 4h15m     # 4 hours 15 minutes
./blockblox set 0         # no limit
```

## Credentials

Credentials are extracted from Chrome and stored in `~/.blockblox.env` with 0600 permissions.

Run `blockblox init` again if your Roblox session expires.

## Assumptions

- Roblox does not have a proper API for screen time controls. This tool uses an undocumented internal API that may break at any time.
- Roblox parental controls do not permit setting screen time on behalf of teens, so execution must come from the teen's own account.

## Build

Requires Go 1.21+.

```
make
```
