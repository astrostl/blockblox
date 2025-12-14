# Claude Code Notes

## API Exploration

When exploring Roblox APIs, use curl for testing rather than modifying Go code.

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
