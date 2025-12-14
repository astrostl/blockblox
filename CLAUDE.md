# Claude Code Notes

## Privacy

Never expose sensitive information in code, configs, or examples:
- Use `~` instead of full paths (e.g., `~/Library/...` not `/Users/username/...`)
- Use placeholders for user IDs (e.g., `USER_ID` not actual IDs)
- Never include real Roblox usernames or display names in examples
- Use generic names like "Alex" or "CoolPlayer123" in documentation

## API Exploration

When exploring Roblox APIs, use curl or Chrome DevTools MCP for testing rather than modifying Go code.

### Tools Available

- **curl**: For testing known endpoints. Use the bash pattern below for authentication.
- **Chrome DevTools MCP**: Installed and configured in `.mcp.json`. Can capture network requests, interact with pages, and discover new endpoints.

**Important**: Always ask before using Chrome DevTools MCP, as it interacts with the user's browser.

### When to Use Each Tool

| Task | Tool |
|------|------|
| Test a known endpoint | curl |
| Discover new/undocumented endpoints | Chrome DevTools MCP |
| Capture request/response from UI action | Chrome DevTools MCP |
| Quick API validation | curl |

### Discovering New Endpoints with Chrome DevTools MCP

Use `mcp__chrome-devtools__list_network_requests` and `mcp__chrome-devtools__get_network_request` to capture API calls made by the Roblox UI.

### Manual Discovery (Alternative)

1. Open Chrome DevTools: `Cmd+Option+I` (Mac) or `F12` (Windows/Linux)
2. Go to **Network** tab
3. Filter by **Fetch/XHR** to see API calls only
4. Perform the action in Roblox UI (e.g., click "Add more time")
5. Find the request and examine:
   - **Headers** tab: URL, method, request headers
   - **Payload** tab: Request body (JSON)
   - **Response** tab: Response body

This is useful for finding undocumented endpoints like screen time extensions.

The cookie values contain special characters (parentheses, ampersands) that break shell parsing. Use this pattern to avoid issues:

```bash
bash -c '
ROBLOX_SECURITY=$(head -1 ~/.blockblox.env | cut -d= -f2-)
ROBLOX_BROWSER_TRACKER=$(tail -1 ~/.blockblox.env | cut -d= -f2-)
curl -s "https://apis.roblox.com/..." \
  -H "Cookie: .ROBLOSECURITY=$ROBLOX_SECURITY; RBXEventTrackerV2=$ROBLOX_BROWSER_TRACKER"
' | jq .
```

Key APIs:
- User info: `https://users.roblox.com/v1/users/authenticated`
- Screen time settings: `https://apis.roblox.com/user-settings-api/v1/user-settings/settings-and-options`
- Weekly consumption: `https://apis.roblox.com/parental-controls-api/v1/parental-controls/get-weekly-screentime?userId=USER_ID`

## Build

```bash
make        # build
make clean  # clean
```

## Documentation

When discovering new Roblox API endpoints or behaviors, update `API.md` with the findings.

## Known Behaviors

When the user exceeds their screen time limit, most Roblox APIs return 403 with "User is moderated". The CLI detects this via `usermoderation.roblox.com/v2/not-approved` (source=2 indicates screen time) and shows a helpful message directing users to `blockblox temp`.

## Code Style

Only use shell commands (bash, zsh) or Go code. Do not create helper scripts in Python, JavaScript, or other languages.

## Testing

When testing `blockblox temp`, only add 1 minute at a time to avoid wasting the user's screen time allowance.
