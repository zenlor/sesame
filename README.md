# Run commands using parameters stored in SSM

# USAGE

```
Usage:
    sesame [--verbose][--human][--configuration CONFIG_FILE] -- <CMD>

Options:
  -V, --version                     print version and exit
  -v, --verbose                     Set debug output
  -H, --human                       Use humanly read-able logs, default is JSON
  -c, --configuration CONFIG_FILE   configuration file to read

AWS environment variables:

- AWS_PROFILE
- AWS_REGION
- AWS_ACCESS_KEY
- AWS_SECRET_KEY

Example:
    sesame -c /config.toml -- bash -c env
    sesame -v -- python main.py
```

# Containers' Entrypoint

```docker
FROM scratch
COPY sesame /sesame
COPY sesame.toml /sesame.toml
COPY app /app
ENTRYPOINT ["/sesame", "-c", "/sesame.toml", "--"]
CMD ["/app", "-x"]
```

# Configuration file

> **important**: All `prefixes` and `secrets` will be expanded with the current
> environment.

In the example below, the command has access to `$PROJECT`, `$ENVIRONMENT` and
`$SERVICE_NAME` environment variables. The resulting calls to the AWS API will
be, for example, `proj/production/web-api-01`.

```toml
prefixes = [
    "$PROJECT/$ENVIRONMENT/$SERVICE_NAME"
]
secrets = [
    "$PROJECT/$ENVIRONMENT/$SERVICE_NAME/DB_USER",
    "$PROJECT/$ENVIRONMENT/$SERVICE_NAME/DB_PASSWORD",
    "$PROJECT/$ENVIRONMENT/$SERVICE_NAME/API_SECRET",
    "$PROJECT/$ENVIRONMENT/$SERVICE_NAME/API_CERTIFICATE",
]

[[rename]]
from = 'DB_USER'
to = 'USERNAME'
```

## prefixes

The list of prefixes that the application will query on SSM Parameter Store. 


## rename

List of variables to rename `from` the incoming name, `to` a new name. In the
previous example the secret `DB_USER` is renamed `USERNAME`

# Usage in containers

Import the sesame container, build your application, take the `/sesame` binary
file and, finally, prepare your build and runtime containers.

```Containerfile
FROM ghcr.io/zenlor/sesame:latest AS sesame
FROM golang:1.22-alpine AS build
ADD . .
RUN go build -o /app
FROM scratch
COPY --from=sesame /sesame /sesame
COPY --from=build /app /app
ADD sesame.toml /sesame.toml
ENTRYPOINT ["/sesame","-c","/sesame.toml","--"]
CMD ["/app"]
```

# LICENSE

[MIT-0](./LICENSE)
