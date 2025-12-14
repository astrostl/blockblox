# Roblox APIs

## Moderation Behavior

When a user is banned or has exceeded their screen time limit, most APIs return `403 Forbidden` with `{"errors":[{"code":0,"message":"User is moderated"}]}`.

**APIs that work when moderated:**
- `usermoderation.roblox.com/v2/not-approved` - Returns restriction status
- `usermoderation.roblox.com/v1/not-approved` - Returns ban details (bans only)
- `parental-controls/add-temporary-screentime` - Works even when screen time blocked
- `users.roblox.com/v1/users/{userId}` - Public API, no auth required

---

## Authentication

Requires two cookies:
- `.ROBLOSECURITY` - Main session cookie (long-lived)
- `RBXEventTrackerV2` - Contains `browserid` for request tracking

And one header for POST requests:
- `X-Csrf-Token` - Obtain by making any POST request without it; token returned in `x-csrf-token` response header

### Getting Cookies from Chrome

1. Log into [roblox.com](https://www.roblox.com) in Chrome
2. Open DevTools: `Cmd+Option+I` (Mac) or `F12` (Windows/Linux)
3. Go to **Application** tab
4. In the left sidebar, expand **Storage > Cookies > https://www.roblox.com**
5. Find and copy the values for:
   - `.ROBLOSECURITY`
   - `RBXEventTrackerV2`

---

## User Settings API

Base URL: `https://apis.roblox.com/user-settings-api/v1`

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

---

## Users API

Base URL: `https://users.roblox.com/v1`

### Get Authenticated User

```
GET /users/authenticated
```

Returns the currently authenticated user's info.

**Note:** Fails with "User is moderated" when banned or screen time blocked.

#### Response

```json
{
    "id": 1234567890,
    "name": "Username",
    "displayName": "Display Name"
}
```

### Get User by ID (Public)

```
GET /users/{userId}
```

Returns user info by ID. **No authentication required.**

#### Response

```json
{
    "id": 1234567890,
    "name": "Username",
    "displayName": "Display Name",
    "description": "User bio...",
    "created": "2020-01-01T00:00:00.000Z",
    "isBanned": false,
    "hasVerifiedBadge": false
}
```

---

## Parental Controls API

Base URL: `https://apis.roblox.com/parental-controls-api/v1`

### Get Weekly Screen Time

```
GET /parental-controls/get-weekly-screentime?userId={userId}
```

Returns screen time consumption for the past 7 days. Always returns exactly 7 entries (days 0-6).

### Add Temporary Screen Time

```
POST /parental-controls/add-temporary-screentime
```

Adds temporary screen time. **Works even when the user is locked out due to exceeding their limit.**

#### Request Headers

| Header | Description |
|--------|-------------|
| `X-Csrf-Token` | CSRF token (obtain from any failed POST request's response header) |
| `Content-Type` | `application/json` |

#### Request Body

```json
{"minutes": 5}
```

#### Response

- `204 No Content` on success (no response body)

**Note:** There is no known GET endpoint to query remaining temporary screen time. The temp time is tracked server-side and only affects whether the user gets blocked.

#### Rate Limits

| Header | Value |
|--------|-------|
| `X-Ratelimit-Limit` | 5 requests per 60 seconds |

---

## User Moderation API

Base URL: `https://usermoderation.roblox.com`

### Get Restriction Status

```
GET /v2/not-approved
```

Returns the current restriction status for the authenticated user.

#### Response (No Restriction)

```json
{"restriction": null}
```

#### Response (Screen Time Blocked)

```json
{
  "restriction": {
    "source": 2,
    "moderationStatus": 2,
    "startTime": "2025-12-14T21:52:17.626Z",
    "endTime": "2025-12-15T07:00:00Z",
    "durationSeconds": 32862
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `source` | Integer | Restriction source (1 = ban, 2 = screen time) |
| `moderationStatus` | Integer | Status code (2 = blocked) |
| `startTime` | String | ISO 8601 timestamp when restriction started |
| `endTime` | String | ISO 8601 timestamp when restriction ends (null if permanent) |
| `durationSeconds` | Integer | Seconds until restriction ends (null if permanent) |

### Get Ban Details

```
GET /v1/not-approved
```

Returns detailed ban information. Only returns data for bans (source=1), not screen time blocks.

#### Response (Banned)

```json
{
  "punishedUserId": 1234567890,
  "messageToUser": "Reason for the ban...",
  "punishmentTypeDescription": "Ban 3 Days",
  "beginDate": "2025-12-14T19:33:50.502Z",
  "endDate": "2025-12-17T19:33:50.502Z",
  "badUtterances": [
    {
      "labelTranslationKey": "Label.AbuseType.Harassment",
      "utteranceText": "offensive message"
    }
  ]
}
```

#### Response (Not Banned)

```json
{}
```

---

## Parental Controls API (continued)

#### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `userId` | Integer | The user ID to query |

#### Response

```json
{
    "dailyScreentimes": [
        {
            "daysAgo": 0,
            "minutesPlayed": 120
        },
        {
            "daysAgo": 1,
            "minutesPlayed": 90
        }
    ],
    "localDayOfWeek": 0
}
```

| Field | Type | Description |
|-------|------|-------------|
| `daysAgo` | Integer | 0 = today, 1 = yesterday, etc. |
| `minutesPlayed` | Integer | Minutes played that day |
| `localDayOfWeek` | Integer | 0 = Sunday, 1 = Monday, etc. |
