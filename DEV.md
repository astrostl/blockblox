# Development Setup

## Chrome DevTools MCP

The project uses Chrome DevTools MCP to capture network requests from the Roblox UI for API discovery.

### Prerequisites

Chrome remote debugging requires a separate user data directory (Chrome limitation).

### Setup

1. Create a Chrome alias with remote debugging:
   ```bash
   alias rchrome='/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --remote-debugging-port=9222 --user-data-dir="$HOME/.chrome-claude"'
   ```

2. Add to `~/.zshrc` to persist the alias.

3. Launch Chrome with `rchrome` and log into Roblox (one-time setup).

4. The MCP config in `.mcp.json` connects to `http://127.0.0.1:9222`.

### Usage

With Chrome running via `rchrome`, Claude Code can:
- List open pages
- Capture network requests
- Interact with page elements
- Discover undocumented API endpoints

### Troubleshooting

**"DevTools remote debugging requires a non-default data directory"**
- You must use `--user-data-dir` pointing to a directory other than `~/Library/Application Support/Google/Chrome`

**MCP can't connect**
- Verify Chrome is running with the debug flag: `curl http://127.0.0.1:9222/json/version`
- Restart Claude Code after changing `.mcp.json`
