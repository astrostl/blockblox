# Changelog

All notable changes to this project will be documented in this file.

## [v0.2.1] - 2025-12-14

### Fixed
- Help text wording for `temp` command

## [v0.2.0] - 2025-12-14

### Added
- `temp` command to add temporary screen time (works even when screen time exceeded)
- Ban detection with reason and time remaining
- Screen time block detection with reset time
- `init` now automatically runs `get` after extracting credentials

### Changed
- User info now displays even when banned or screen time blocked
- "Temporary time active" status shown when over limit but not blocked
- Raw minutes only shown when duration includes hours or days

### Fixed
- Graceful handling of moderated accounts instead of raw API errors

## [v0.1.5] - 2025-12-14

### Added
- Show "Remaining: Unlimited" when no limit is set
- Show remaining time when under limit
- Show consumption in `set` command with warning when over limit
- Usage examples in README

## [v0.1.4] - 2025-12-14

### Changed
- Show both display name and username in output

### Added
- Instructions for getting cookies from Chrome in api.md
- Documentation for Users and Parental Controls APIs

## [v0.1.3] - 2025-12-14

### Added
- Show today's consumption in `get` command
- Note about `go install` version behavior

## [v0.1.2] - 2025-12-14

### Fixed
- Module path for `go install` compatibility

## [v0.1.1] - 2025-12-14

### Added
- `--version` flag
- Installation instructions in README

### Fixed
- Homebrew install instructions

## [v0.1.0] - 2025-12-14

### Added
- Initial release
- `init` command to extract credentials from Chrome
- `get` command to show current screen time limit
- `set` command to update screen time limit
- Homebrew tap for installation
- API documentation
