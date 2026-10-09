# Roadmap

Release strategy and branch policy for aurl (as of 2026-10).

## Current status

| Channel | Version | Contents | Branch |
|---|---|---|---|
| brew `aurl` | 1.1.6 | v1 | `v1-maintenance` |
| brew `aurl-v2-alpha` | 2.0.0-alpha20251208b | v1 CLI + tokens stored in the OS keychain (PR #20) | `v2-alpha-maintenance` |
| (unreleased) | - | v2 rewrite (`aurl add` / `aurl exec`, config at `~/.aurl/config`, secrets in the keyring) | `master` |

## Branches

| Branch | Purpose | Rules |
|---|---|---|
| `master` | v2 development. Will be released as brew `aurl` | All new features and fixes land here first |
| `v2-alpha-maintenance` | Maintenance of the current alpha (`aurl-v2-alpha`) | Cherry-picked fixes only. Frozen once an alpha is released from `master` |
| `v1-maintenance` | Maintenance of v1 | Maintenance only, no other work |

- Merge fixes into `master` first, then `git cherry-pick -x` them to other branches as needed (e.g. the proxy fix from PR #22).
- Releases are triggered by pushing a tag, regardless of branch:
  - `v2.0.0-alpha*` → `.goreleaser.v2-alpha.yml` → brew `aurl-v2-alpha`
  - anything else → `.goreleaser.yaml` → brew `aurl`
  - Nothing changes on the brew side until a tag is pushed, so progress on `master` does not affect existing users.

## v2 release plan

- Release v2 on brew as `aurl`, replacing v1.
  - `brew upgrade` switches users to v2, and the command changes to `aurl add` / `aurl exec` (v1-style `aurl [-p profile] <url>` stops working).
  - Homebrew does not distinguish major versions, so users may be upgraded to v2 unintentionally. This is mitigated by the following.
- Provide a v1 → v2 migration path with `aurl migrate`.
- For users who want to keep using v1, release brew `aurl-v1` from `v1-maintenance`.

### Breaking changes from v1 to v2

| Item | v1 | v2 |
|---|---|---|
| Command | `aurl [-p profile] <url>` (defaults to `default`) | `aurl exec <profile> <url>` (profile required) |
| Config file | `~/.aurl/profiles` | `~/.aurl/config` |
| client_id / client_secret / username / password | Plain text in the config file | Keyring (`aurl add`) |
| Key names | `default_content_type` / `default_user_agent` | `content_type` / `user_agent` |
| Default Content-Type | None | `application/json` |
| Tokens | `~/.aurl/token/*.json` (keychain in the alpha) | Keyring (re-authentication required) |

### TODO before the v2 release

1. Fix the build
   - [ ] With `CGO_ENABLED=0`, the build fails on all OSes because of the 1Password SDK (`onepassword-sdk-go`).
   - [ ] CI still uses Go 1.19 (go.mod requires 1.25).
   - [ ] The macOS keychain backend requires cgo (`//go:build darwin && cgo`), so the current macOS build cannot use the keychain.
2. Migration support
   - [ ] `aurl migrate`: read `~/.aurl/profiles`, write settings to `~/.aurl/config` and secrets to the keyring. Apply v1 defaults and handle renamed keys. Leave the v1 file untouched.
   - [ ] When invoked in v1 style (`-p` or `aurl <url>`), point users to `aurl exec` / `aurl migrate`.
   - [ ] When `exec` cannot find the profile and `~/.aurl/profiles` exists, suggest `aurl migrate`.
   - [ ] Use brew caveats to announce the breaking changes, `aurl migrate`, and `aurl-v1`.
   - [ ] Rewrite the README for v2 and add a migration guide.
3. Other
   - [ ] `vault.ConfigFile.ProfileSections()` drops every section except `default` (affects shell completion).
   - [ ] Check whether the alpha (zalando/go-keyring) and v2 (byteness/keyring) read/write the same keyring item (service `aurl`, key = profile name).
4. Release
   - [ ] Release an alpha from `master` and let alpha users try the migration → freeze `v2-alpha-maintenance`.
   - [ ] Change the release config on `v1-maintenance` to publish brew `aurl-v1`. (It currently publishes brew `aurl`; tagging v1 after the v2 release would revert `aurl` to v1.)
   - [ ] Release `aurl-v1`.
   - [ ] Release `v2.0.0` (brew `aurl` switches to v2).

## Open questions

- Should `aurl-v1` install its binary as `aurl-v1` or `aurl`? (`aurl` keeps existing scripts working but conflicts with brew `aurl`.)
- Should the module path in go.mod become `/v2`? (Otherwise `go install ...@latest` does not install v2.)
- What to do with old PRs: #19 (duplicates v2's `aurl add`), #21 (dependabot).
