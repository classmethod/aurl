# aurl

![Release](https://github.com/classmethod/aurl/workflows/Release/badge.svg)
[![License](https://img.shields.io/github/license/classmethod/aurl.svg)](https://github.com/classmethod/aurl/blob/master/LICENSE)

## Description

HTTP CLI client with OAuth 2.0 authentication.

You know `curl` is powerful command line tool and you can make any complex HTTP request to every servers.
But the target web server is secured by OAuth 2.0, you must send another HTTP request to the authorization server
before making principal request. And more, you should to manage issued access tokens for every resources.

`aurl` is a command-line tool that process OAuth 2.0 dance and manage access/refresh tokens automatically.
Client credentials and tokens are stored in your OS keyring or 1Password, not in plain-text files.

**Note:** Currently, `aurl` is not support OAuth 1.0a. Your pull-request is appreciated.

This README describes aurl v2. v2 is not compatible with v1 on the command line and in the config file.
See [Migrating from v1](#migrating-from-v1).

## Install

### homebrew

```bash
brew tap classmethod/repos
brew install aurl
```

If you want to keep using v1, install `aurl-v1` instead:

```bash
brew install aurl-v1
```

`aurl-v2-alpha` conflicts with `aurl`. If you have it installed, switch with `brew uninstall aurl-v2-alpha && brew install aurl`.

### manually

Download the [pre-compiled binaries](https://github.com/classmethod/aurl/releases) from GitHub Releases.

### from source

Requires Go 1.26.10 or later (see `go.mod`).

```bash
git clone https://github.com/classmethod/aurl.git
cd aurl
make            # or: go build -o bin/aurl
bin/aurl --version
```

## Quick start

```bash
$ aurl add                                # register a profile interactively
Enter Profile Name: myapi
Select Grant Type:
...
$ aurl myapi https://api.example.com/users
...http.response.body...
$ aurl myapi /users                       # if base_url is set for the profile
...http.response.body...
```

## Usage

### Making requests

| command                     | behavior                                                                      |
| --------------------------- | ----------------------------------------------------------------------------- |
| `aurl <profile> <url\|path>` | Send a request with the access token of `<profile>`.                        |
| `aurl <profile>`            | Prompt for the URL. The prompt shows the profile's `base_url` if it is set.   |
| `aurl`                      | Select a profile from a list, then prompt for the URL.                        |

An argument that does not start with `http://` or `https://` is treated as a path and appended to the profile's `base_url`
(simple concatenation, not RFC 3986 resolution: `base_url = https://api.example.com/v1` + `/users` → `https://api.example.com/v1/users`).
A path without `base_url` is an error.

Interactive prompts need a terminal, so in scripts and CI always pass both: `aurl <profile> <url>`.

aurl sends the access token in the `Authorization: Bearer` header. When there is no valid token, aurl refreshes it with
the refresh token if one is available, otherwise it runs the grant flow of the profile. Tokens are saved in the keyring.
If the API returns 401, aurl exits with status 1 without printing the body and keeps the stored token; if the token was
revoked on the server side, use `--renew-token`.

By default aurl prints the response body to stdout. Errors are printed to stderr.
aurl honors the `HTTPS_PROXY` / `HTTP_PROXY` / `NO_PROXY` environment variables.

```bash
$ aurl myapi -X POST /users -d '{"name":"john"}'
...http.response.body...
$ aurl myapi --no-print-body --print-headers -X OPTIONS /users
{"Content-Type":["application/json;charset=UTF-8"],"Date":["Tue, 17 Feb 2015 08:16:41 GMT"], ...}
```

`aurl <profile> <url>` is a shortcut for `aurl curl <profile> <url>`. `aurl help curl` shows the request flags.

### Request flags

| flag                           | description                                                                  |
| ------------------------------ | ---------------------------------------------------------------------------- |
| `-X`, `--request=METHOD`       | HTTP request method (default: `GET`)                                         |
| `-H`, `--header=HEADER:VALUE`  | Add an HTTP header. Repeatable. Overrides the profile's `content_type` / `user_agent` |
| `-d`, `--data=DATA`            | HTTP request body                                                            |
| `-k`, `--insecure`             | Disable SSL certificate verification                                         |
| `--[no-]print-body`            | Print the response body to stdout (default: enabled)                         |
| `--[no-]print-headers`         | Print the response headers as JSON to stdout (default: disabled)             |
| `--renew-token`                | Ignore the stored token and run the grant flow again                         |

### Global flags

| flag                                  | environment variable | description                                       |
| ------------------------------------- | -------------------- | ------------------------------------------------- |
| `-v`, `--verbose`                     |                      | Verbose logging to stderr (includes HTTP dumps, see below) |
| `--version`                           |                      | Show the version                                  |
| `--backend=BACKEND`                   | `AURL_BACKEND`       | Secret backend. See [Secret backends](#secret-backends) |
| `--op-vault-id` and other `--op-*`    | `AURL_OP_*`          | 1Password settings                                |

`-v` output contains the access token, the client secret (Basic auth of token requests), the password of the password grant
and the token response as is, so be careful when sharing it or pasting it into an issue.

The v1 flags `-p` / `--profile`, `-V` (use `--version`) and `-h` (use `--help`) no longer exist.

### Selection list

`aurl` (profile selection) and `aurl add` (grant type selection) show a selection list:

| key                      | action                       |
| ------------------------ | ---------------------------- |
| printable ASCII character | filter the list (case-insensitive substring match) |
| Backspace                | delete the last filter character |
| ↑ / ↓, Ctrl-P / Ctrl-N   | move the cursor              |
| Enter                    | select                       |
| Ctrl-C / Ctrl-D          | cancel                       |

When stdin or stderr is not a terminal (and on Windows), a numbered list is shown and you enter the number instead.

### Adding a profile

```bash
$ aurl add
```

`aurl add` asks for the profile name, the grant type and the settings required for it, then writes the settings to
`~/.aurl/config` and the client ID / client secret (and username / password for the password grant) to the keyring.
Content type and user agent default to `application/json` and `aurl` when left empty.

`aurl add` fails if the profile already exists. Use `aurl add --force` to overwrite it (you re-enter every setting).
Adding a profile also removes its stored token.

There is no command to remove a profile. Delete its section from `~/.aurl/config` and the keyring entries `<profile>` and
`<profile>-TokenInfo` by hand.

### Reserved names

The following names are reserved for subcommands and cannot be used as profile names:

```
add, migrate, help, curl, exec, run, list, remove, clear, login, export, rotate, proxy, token
```

`aurl add` rejects them and `aurl migrate` skips them. If your config already has a profile with a reserved name,
call it with `aurl curl <name> <url>`.

## Configuration

### Config file

Profile settings are stored in `~/.aurl/config` ([INI file](http://en.wikipedia.org/wiki/INI_file)).
Set `AURL_CONFIG_FILE` to use another path. The file is created when it does not exist.
Each section name is used as the profile name (case-sensitive). A section name containing `.` is a child section and
inherits missing keys from its parent (e.g. `[myapi.staging]` from `[myapi]`), so be careful when using `.` in profile names.

Create profiles with `aurl add`: the client ID / secret are stored in the keyring, so a profile written only in the config
file fails with "Failed to get credentials". You can edit the config afterwards (e.g. to add `base_url`).

| key name                   | description                                    | default            |
| -------------------------- | ---------------------------------------------- | ------------------ |
| grant_type                 | `authorization_code`, `implicit`, `password` or `client_credentials` | (none, required) |
| auth_server_auth_endpoint  | OAuth2 authorization endpoint URI              | (none)             |
| auth_server_token_endpoint | OAuth2 token endpoint URI                      | (none)             |
| redirect                   | redirect URI                                   | (none)             |
| scopes                     | space separated scope values                   | (none)             |
| content_type               | `Content-Type` header sent with every request  | `application/json` |
| user_agent                 | `User-Agent` header                            | `aurl`             |
| base_url                   | base URL used when a path is given             | (none)             |

Required keys per grant type:

| grant_type           | required keys                                               | secrets used (keyring)          |
| -------------------- | ----------------------------------------------------------- | ------------------------------- |
| `authorization_code` | auth_server_auth_endpoint, auth_server_token_endpoint, redirect | client ID, client secret    |
| `implicit`           | auth_server_auth_endpoint, redirect                         | client ID                       |
| `password`           | auth_server_token_endpoint                                  | client ID, client secret, username, password |
| `client_credentials` | auth_server_token_endpoint                                  | client ID, client secret        |

For `authorization_code`, aurl opens the authorization URL in the browser and asks for the authorization code.
For `implicit`, aurl opens the authorization URL and asks you to paste the token response as JSON
(e.g. `{"access_token":"...","token_type":"Bearer","expires_in":3600}`).

```ini
[myapi]
grant_type = authorization_code
auth_server_auth_endpoint = https://auth.example.com/oauth/authorize
auth_server_token_endpoint = https://auth.example.com/oauth/token
redirect = https://auth.example.com/oauth/oob
scopes = read write
base_url = https://api.example.com

[batch]
grant_type = client_credentials
auth_server_token_endpoint = https://auth.example.com/oauth/token
scopes = read
content_type = application/x-www-form-urlencoded
```

### Secrets and tokens

`client_id`, `client_secret`, `username` and `password` are not read from the config file.
They are stored in the keyring under the profile name by `aurl add` or `aurl migrate`.
Access / refresh tokens are stored in the keyring under `<profile>-TokenInfo`.

## Secret backends

Select the backend with `--backend` or `AURL_BACKEND`. The default is the OS keyring.
`aurl --help` lists the backends available on your platform.

| backend          | description                         | required settings                                                         |
| ---------------- | ----------------------------------- | ------------------------------------------------------------------------- |
| `keychain`       | macOS Keychain (default on macOS)   |                                                                           |
| `wincred`        | Windows Credential Manager (default on Windows) |                                                               |
| `secret-service` | Secret Service, e.g. GNOME Keyring (default on Linux) |                                                         |
| `op`             | 1Password Service Account           | `--op-vault-id`, `AURL_OP_SERVICE_ACCOUNT_TOKEN` (prompted if not set)    |
| `op-connect`     | 1Password Connect server            | `--op-vault-id`, `--op-connect-host`, `AURL_OP_CONNECT_TOKEN` (prompted if not set) |
| `op-desktop`     | 1Password desktop app integration   | `--op-vault-id`, `--op-desktop-account-id`                                |

| flag                      | environment variable         | description                                               |
| ------------------------- | ---------------------------- | --------------------------------------------------------- |
| `--op-vault-id`           | `AURL_OP_VAULT_ID`           | UUID of the 1Password vault                               |
| `--op-connect-host`       | `AURL_OP_CONNECT_HOST`       | 1Password Connect server HTTP(S) URI                      |
| `--op-desktop-account-id` | `AURL_OP_DESKTOP_ACCOUNT_ID` | 1Password account name or UUID for the desktop app        |
| `--op-timeout`            | `AURL_OP_TIMEOUT`            | Timeout for 1Password API operations (default: `15s`)     |

1Password items are created with the title prefix `aurl` and the tag `aurl`.

To use `op-desktop`, enable "Integrate with other apps" in the Developer settings of the 1Password app.
`op-desktop` does not work with the pre-compiled Linux binaries because they are built without cgo.

```bash
$ export AURL_OP_VAULT_ID=xxxxxxxxxxxxxxxxxxxxxxxxxx
$ export AURL_OP_DESKTOP_ACCOUNT_ID=my.1password.com
$ aurl --backend op-desktop myapi https://api.example.com/path/to/resource
```

## Migrating from v1

1. Run `aurl migrate`. It reads `~/.aurl/profiles` (or `--profiles-file=PATH`), writes the settings to `~/.aurl/config`
   and the secrets to the keyring.
   - v1 defaults are applied to missing keys (`grant_type = authorization_code`, `scopes = root`,
     client ID / secret `aurl`), and `default_content_type` / `default_user_agent` are renamed.
   - Profiles that already exist in `~/.aurl/config` are skipped. Use `aurl migrate --force` to overwrite them.
   - Profiles with a [reserved name](#reserved-names) are skipped. Rename them in `~/.aurl/profiles` and re-run.
2. Check that your profiles work: `aurl <profile> <url>` (the v1 `default` profile is now `aurl default <url>`).
   Tokens are not migrated, so you have to authenticate again.
3. `~/.aurl/profiles` is left untouched. It contains plain-text secrets, so delete it once you have verified the migration.
   `~/.aurl/token/*.json` is not used by v2 and contains plain-text tokens, so you can delete it as well.

### Breaking changes from v1 to v2

| Item | v1 | v2 |
|---|---|---|
| Command | `aurl [-p profile] <url>` (defaults to `default`) | `aurl <profile> <url>` (profile required) |
| Config file | `~/.aurl/profiles` | `~/.aurl/config` |
| client_id / client_secret / username / password | Plain text in the config file | Keyring (`aurl add`) |
| Key names | `default_content_type` / `default_user_agent` | `content_type` / `user_agent` |
| Default Content-Type | None | `application/json` |
| Default User-Agent | `aurl-<version>` | `aurl` |
| Tokens | `~/.aurl/token/*.json` (keychain in the alpha) | Keyring (re-authentication required) |

## Contribution

1. Fork ([https://github.com/classmethod/aurl/fork](https://github.com/classmethod/aurl/fork))
1. Create a feature branch named like `feature/something_awesome_feature` from the `master` branch
1. Commit your changes
1. Rebase your local changes against the `master` branch
1. Run test suite with the `go test ./...` command and confirm that it passes
1. Run `gofmt -s`
1. Create new Pull Request against the `master` branch

## Build

```bash
$ make deps
$ make
```

## Author

[Daisuke Miyamoto](https://github.com/dai0304)

## Maintainer

[Seiichi Arai](https://github.com/seiichi1101)
