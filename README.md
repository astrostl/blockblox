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
blockblox init

# Get current screen time limit and consumption
blockblox get

# Set screen time limit
blockblox set 90        # 90 minutes
blockblox set 90m       # 90 minutes
blockblox set 4h        # 4 hours
blockblox set 4h15m     # 4 hours 15 minutes
blockblox set 0         # no limit
```

### Examples

**Check status (no limit):**
```
$ blockblox get
User: Alex (@CoolPlayer123)
Limit: No limit (0 minutes)
Consumed: 2 hour(s) 30 minute(s) (150 minutes)
Remaining: Unlimited
```

**Check status (with limit):**
```
$ blockblox get
User: Alex (@CoolPlayer123)
Limit: 4 hour(s) (240 minutes)
Consumed: 2 hour(s) 30 minute(s) (150 minutes)
Remaining: 1 hour(s) 30 minute(s)
```

**Set limit (over limit warning):**
```
$ blockblox set 2h
User: Alex (@CoolPlayer123)
Limit set to: 2 hour(s) (120 minutes)
Consumed: 2 hour(s) 30 minute(s) (150 minutes)
⚠️  Over limit by 30 minute(s)
```

## Credentials

Credentials are extracted from Chrome and stored in `~/.blockblox.env` with 0600 permissions.

If your Roblox session expires, log out and log back in using Chrome, then run `blockblox init` again.

## Assumptions

- Roblox does not have a proper API for screen time controls. This tool uses an undocumented internal API that may break at any time.
- Roblox parental controls do not permit setting screen time on behalf of teens, so execution must come from the teen's own account.

## Roadmap

- Scheduling support (e.g., different limits for weekdays/weekends)
- Limit and consumption history graphing

## Build

Requires Go 1.21+.

```
make
```
