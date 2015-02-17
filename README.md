aurl
====

## Description

HTTP CLI client with OAuth2 authentication

curl is powerful command line tool and you can make any complex HTTP request to every servers.  But the target web
server is secured by OAuth2, you must request another HTTP request to the authorization server before making
principal request.  And more, you should to manage issued access tokens for every resources.

aurl is auto processing OAuth dance and manage access/refresh tokens.

## Install

To install, use `go get`:

```bash
$ go get -d github.com/classmethod-aws/aurl
```

## Usage

### Profile configuration

First, you MUST create profile setting file `~/.aurl/profiles` file which format is described below.
Profile setting file format is typically called [INI file](http://en.wikipedia.org/wiki/INI_file).
Each section name is used as profile name.

```
[default]
auth_server_auth_endpoint = https://api.example.com/oauth/authorize
auth_server_token_endpoint = https://api.example.com/oauth/token
redirect = https://api.example.com/oauth/oob

[foobar]
grant_type = password
auth_server_token_endpoint = https://api.example.com/oauth/token
username = john
password = pass1234
```

### Token store file

Token store file `~/.aurl/tokens` is used by acurl internally.  Retrieved access/refresh token is stored in this file.
You SHOULD NOT edit this file manually because this file is overwritten every time curl is executed.
You MAY lose comment and another extra data.

Just for information, token sotore file example is following:

```
[default]
expiry = 1424049169
access_token = xxxx
token_type = bearer
refresh_token = yyyy

[foobar]
expiry = 1424141030
access_token = zzzz
token_type = bearer
```

### Execution

###### SYNOPSIS

```bash
$ aurl [global options] command [command options] [arguments...]
```

`command` is every http method (e.g. `get`, `post`, `delete`) and first argument is target url.

###### EXAMPLE

```bash
$ aurl get http://api.example.com/path/to/resource
$ aurl post http://api.example.com/path/to/resource --data "foobar"
```

aurl make request with access token in `Authorization` header of `default` profile.
You can specify profile name with `--profile` option.

```bash
$ aurl --profile foobar get http://api.example.com/path/to/resource
```

By default aurl prints response body in stdout.  When an error occured the detail is printed in stderr.
You may want not response body but response header, then you can use `--no-body` and `--print-headers` option.

```bash
$ aurl --no-body --print-headers options http://api.example.com/path/to/resource
{"Content-Type":["application/json;charset=UTF-8"],"Date":["Tue, 17 Feb 2015 08:16:41 GMT"],"Server":["nginx/1.6.2"], "...": "..."}
```

## Contribution

1. Fork ([https://github.com/classmethod-aws/oauthttp/fork](https://github.com/classmethod-aws/aurl/fork))
1. Create a feature branch named like `feature/something_awesome_feature` from `development` branch
1. Commit your changes
1. Rebase your local changes against the `develop` branch
1. Run test suite with the `go test ./...` command and confirm that it passes
1. Run `gofmt -s`
1. Create new Pull Request

## Author

[Daisuke Miyamoto](https://github.com/miyamoto-daisuke)