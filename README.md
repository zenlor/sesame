# Sesame

Run commands with secrets fetched from AWS Systems Manager Parameter Store.

Sesame fetches parameters by path recursively and by explicit name, injects
them as environment variables, then executes the target command. No secrets
written to disk, no scripts to maintain.

# Quick start

```toml
# sesame.toml
prefix = "/myapp/production"
secrets = [
    "/myapp/production/DB_PASSWORD",
    "/myapp/production/API_KEY",
]
```

```sh
sesame -c sesame.toml -- python main.py
```

The application `main.py` will have `API_KEY` and `DB_PASSWORD` available as
environment variables.

# Configuration

## File format

```toml
prefix = "/myapp/staging"
secrets = [
    "/myapp/staging/DB_USER",
    "/myapp/staging/DB_PASSWORD",
]
```

- `prefix`: SSM path. Sesame will fetch all parameters under this path
  recursively.
- `secrets`: explicit list of full parameter names. This is a limitation of the
  AWS APIs.

## Env var expansion

Configuration supports `$VAR` and `${VAR}` syntax for both parameter prefix and
secret names:

```toml
prefix = "$PROJECT/$ENV/$SERVICE"
secrets = [
    "$PROJECT/$ENV/$SERVICE/DB_PASSWORD",
]
```

**NOTE**: If any variable is unset it expands to an empty string.

# Requirements

Sesame uses AWS's SDK default authentication chain.

## IAM policy

The executing principal (via instance role, ECS task role, or explicit keys) requires:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ssm:GetParametersByPath",
        "ssm:GetParameters"
      ],
      "Resource": "arn:aws:ssm:*:*:parameter/myapp/*"
    }
  ]
}
```

`GetParametersByPath` requires `WithDecryption` if parameters are
SecureStrings. The policy above works for both encrypted and unencrypted
parameters.

**IMPORTANT**: The application also requires `kms:Decrypt` on the right key to
decrypt secrets and parameters.

# Command line

```
sesame [flags] -- <CMD> [ARGS...]

Flags:
  -V, --version        print version and exit
  -v, --verbose        debug output (JSON)
  -H, --human          human-readable logs (text)
  -c, --config FILE    config file path (default: sesame.toml)
```

> **Important**
>
> `--` separates sesame flags from the command. Everything after `--` is passed to `execve` unchanged.

# How it works

1. Read and parse the configuration. Expand environment variables in
   secret/parameters keys.
2. Call `GetParametersByPath` recursively under `prefix`.
3. Call `GetParameters` for each item in `secrets` (requests are performed in
   batches of 10 secrets).
4. Expose the values of parameters and secrets as environment variables.
5. Call `syscall.Exec` to execute and handover the environment to the
   application.

Sesame dies when calling the application freeing used resources and giving
control to the child application.

# Security notes

- Secrets travel from AWS to the process env, never to disk.
- The parent process is replaced by the child.
- Access is controlled by IAM alone, applications should have
- If the config file is user-controlled, `$VAR` expansion lets them override any env var present at startup. Restrict file permissions on `sesame.toml`.

# License

[MIT-0](./LICENSE)
