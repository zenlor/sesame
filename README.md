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

All configuration values will be parsed and expanded with the current
environment variables. In the example below, the command has access to
`$PROJECT`, `$ENVIRONMENT` and `$SERVICE_NAME` environment variables. The
resulting calls to the AWS API will be, for example,
`proj/production/web-api-01`

```toml
prefix = "$PROJECT/$ENVIRONMENT/$SERVICE_NAME"
secrets = [
    "$PROJECT/$ENVIRONMENT/$SERVICE_NAME/DB_USER",
    "$PROJECT/$ENVIRONMENT/$SERVICE_NAME/DB_PASSWORD",
    "$PROJECT/$ENVIRONMENT/$SERVICE_NAME/API_SECRET",
    "$PROJECT/$ENVIRONMENT/$SERVICE_NAME/API_CERTIFICATE",
]
```

# LICENSE

[MIT-0](./LICENSE)
