# Roblox User Settings API

Base URL: `https://apis.roblox.com/user-settings-api/v1`

## Authentication

Requires two cookies:
- `.ROBLOSECURITY` - Main session cookie (long-lived)
- `RBXEventTrackerV2` - Contains `browserid` for request tracking

And one header:
- `X-Csrf-Token` - Obtain by making any POST request without it; token returned in `x-csrf-token` response header

## Endpoints

### Get User Settings and Options

```
GET /user-settings/settings-and-options
```

Returns all user settings with their current values and available options.

#### Response

```json
{
    "dailyScreenTimeLimit": {
        "currentValue": 195,
        "options": [
            {
                "option": {
                    "optionType": "Integer"
                },
                "requirement": "SelfUpdateSetting"
            }
        ]
    },
    // ... other settings
}
```

### Screen Time Limit

| Field | Type | Description |
|-------|------|-------------|
| `dailyScreenTimeLimit.currentValue` | Integer | Daily screen time limit in **minutes** |

#### Example Values

| Minutes | Display |
|---------|---------|
| 0 | No limit |
| 60 | 1 hour |
| 120 | 2 hours |
| 195 | 3 hours 15 minutes |

#### Requirements

- `SelfUpdateSetting`: Authenticated user can modify their own setting

### Update User Settings

```
POST /user-settings
```

Updates user settings. Only send the fields you want to change (partial update).

#### Request Headers

| Header | Description |
|--------|-------------|
| `X-Csrf-Token` | CSRF token (obtain from any failed POST request's response header) |
| `Content-Type` | `application/json` |

#### Required Cookies

| Cookie | Description |
|--------|-------------|
| `.ROBLOSECURITY` | Session cookie |
| `RBXEventTrackerV2` | Contains `browserid` parameter |

#### Request Body

```json
{
    "dailyScreenTimeLimit": 195
}
```

Only include the setting(s) you want to update.

#### Response

```json
{"cascadingSettingUpdates":{}}
```

Empty `cascadingSettingUpdates` indicates success.

#### Rate Limits

| Header | Value |
|--------|-------|
| `X-Ratelimit-Limit` | 30 requests per 60 seconds |
