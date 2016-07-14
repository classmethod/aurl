aurl
====

## Description

HTTP CLI client with OAuth 2.0 authentication.

You know `curl` is powerful command line tool and you can make any complex HTTP request to every servers.  But the target web server is secured by OAuth 2.0, you must send another HTTP request to the authorization server before making principal request.  And more, you should to manage issued access tokens for every resources.

`aurl` is a command-line tool that process OAuth 2.0 dance and manage access/refresh tokens automatically.

**Note:** Currently, `aurl` is not support OAuth 1.0a.  Your pull-request is appreciated.

## Install

To install, use `homebrew` or `go get`:

```bash
$ brew tap classmethod/repos
$ brew install aurl
```

```bash
$ go get -d github.com/classmethod/aurl
```

## Usage

### Profile configuration

First, you must create profile setting file `~/.aurl/profiles` file which format is described below.
Profile setting file format is typically called [INI file](http://en.wikipedia.org/wiki/INI_file).
Each section name is used as profile name.

###### SYNOPSIS

Section name is utilized as profile name.  In each section following key settings are available:

| key name                      | description                       | default value | available values | mandatory                       |
| ----------------------------- | --------------------------------- |:-------------:|:----------------:|:-------------------------------:|
| grant\_type                   | OAuth2 grant type                 | authorization_code | authorization_code, password, client_credentials | no |
| client\_id                    | client id                         | aurl          | (any)            | no                              |
| client_secret                 | client secret                     | aurl          | (any)            | no                              |
| auth\_server\_auth\_endpoint  | OAuth2 authorization endpoint URI | (none)        | (any)            | YES (except for password grant) |
| auth\_server\_token\_endpoint | OAuth2 token endpoint URI         | (none)        | (any)            | YES                             |
| redirect                      | redirect URI                      | (none)        | (any)            | YES (except for password grant) |
| scopes                        | space separated scope values      | read write    | (any)            | no                              |
| username                      | username for password grant       | (none)        | (any)            | no (except for password grant)  |
| password                      | password for password grant       | (none)        | (any)            | no (except for password grant)  |
| default\_content\_type        | default content type header       | (none)        | (any)            | no                              |


Implicit flow is not supported currently.

###### EXAMPLE

```
[default]
auth_server_auth_endpoint = https://api.example.com/oauth/authorize
auth_server_token_endpoint = https://api.example.com/oauth/token
redirect = https://api.example.com/oauth/oob
default_content_type = application/json

[foobar]
grant_type = password
client_id = foobar
client_secret = bazqux
auth_server_token_endpoint = https://api.example.com/oauth/token
scopes = read write global
username = john
password = pass1234

[fb]
client_id = your_facebook_App_ID
client_secret = your_facebook_App_Secret
auth_server_auth_endpoint = https://www.facebook.com/dialog/oauth
auth_server_token_endpoint = https://graph.facebook.com/oauth/access_token
redirect = https://www.facebook.com/connect/login_success.html
scopes = public_profile email user_friends

[google]
client_id = xxxxxxxx.apps.googleusercontent.com
client_secret = yyyyyyyy
auth_server_auth_endpoint = https://accounts.google.com/o/oauth2/auth
auth_server_token_endpoint = https://www.googleapis.com/oauth2/v3/token
redirect = urn:ietf:wg:oauth:2.0:oob
scopes = https://www.googleapis.com/auth/plus.login https://www.googleapis.com/auth/userinfo.email

```

### Token store file

Token store file `~/.aurl/token/*.json` is used by aurl internally.  Retrieved token response body is stored in this file.
You SHOULD NOT edit this file manually because this file is overwritten at any time curl is executed.
You may lose comment and another extra data.


### Execution

###### SYNOPSIS

```bash
usage: aurl [<flags>] <url>

Command line utility to make HTTP request with OAuth2.

Flags:
      --help                     Show context-sensitive help (also try --help-long and --help-man).
  -p, --profile="default"        Set profile name. (default: "default")
  -X, --request="GET"            Set HTTP request method. (default: "GET")
  -H, --header=HEADER:VALUE ...  Add HTTP headers to the request.
  -d, --data=DATA                Set HTTP request body.
  -k, --insecure                 Disable SSL certificate verification.
      --print-body               Enable printing response body to stdout. (default: enabled, try --no-print-body)
      --print-headers            Enable printing response headers JSON to stdout. (default: disabled, try --no-print-headers)
  -V, --verbose                  Enable verbose logging to stderr.
      --version                  Show application version.

Args:
  <url>  The URL to request
```

###### EXAMPLE

```bash
$ aurl http://api.example.com/path/to/resource
...http.response.body...
$ aurl -X POST http://api.example.com/path/to/resource --data "foobar"
...http.response.body...
```

aurl make request with access token in `Authorization` header of `default` profile.
You can specify profile name with `--profile` option.

```bash
$ aurl --profile fb https://graph.facebook.com/me
{"id":"...","email": ... }
$ aurl --profile google https://www.googleapis.com/plus/v1/people/me
{
 "kind": "plus#person",
...
}
```

By default aurl prints response body in stdout.  When an error occured the detail is printed in stderr.
You may want not response body but response header, then you can use `--no-print-body` and `--print-headers` option.

```bash
$ aurl --no-print-body --print-headers -X OPTIONS http://api.example.com/path/to/resource
{"Content-Type":["application/json;charset=UTF-8"],"Date":["Tue, 17 Feb 2015 08:16:41 GMT"],"Server":["nginx/1.6.2"], ...}
```

## Contribution

1. Fork ([https://github.com/classmethod/aurl/fork](https://github.com/classmethod/aurl/fork))
1. Create a feature branch named like `feature/something_awesome_feature` from `development` branch
1. Commit your changes
1. Rebase your local changes against the `develop` branch
1. Run test suite with the `go test ./...` command and confirm that it passes
1. Run `gofmt -s`
1. Create new Pull Request

## Author

[Daisuke Miyamoto](https://github.com/dai0304)