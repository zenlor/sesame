package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/pelletier/go-toml/v2"
)

var USAGE string = `Usage:
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

`

type Configuration struct {
	Prefix  string   `toml:"prefix"`
	Secrets []string `toml:"secrets"`
}

var Version string

func main() {
	flag.Usage = func() { fmt.Fprintf(os.Stderr, "%s\n", USAGE) }

	var (
		verboseFlag, versionFlag, humanOutputFlag bool
		configFilenameFlag                        string
	)
	flag.BoolVar(&versionFlag, "version", false, "show the current version")
	flag.BoolVar(&verboseFlag, "v", false, "set verbose output")
	flag.BoolVar(&verboseFlag, "verbose", false, "set verbose output")
	flag.BoolVar(&humanOutputFlag, "human", false, "humanly readable logging")
	flag.BoolVar(&humanOutputFlag, "H", false, "humanly readable logging")
	flag.StringVar(&configFilenameFlag, "config", "sesame.toml", "config file path")
	flag.StringVar(&configFilenameFlag, "c", "sesame.toml", "config file path")
	flag.Parse()

	argv := flag.Args()
	if len(argv) == 0 {
		flag.Usage()
		os.Exit(0)
	}

	slog.Debug("flags",
		"version", versionFlag,
		"human", humanOutputFlag,
		"config", configFilenameFlag,
		"argv", argv,
	)

	if versionFlag {
		slog.Info("sesame", "version", Version)
		os.Exit(0)
	}

	var logHandler slog.Handler
	if humanOutputFlag && verboseFlag {
		logHandler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		})
	} else if humanOutputFlag {
		logHandler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})
	} else if verboseFlag {
		logHandler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		})
	} else {
		logHandler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})
	}
	slog.SetDefault(slog.New(logHandler))

	configData, err := os.ReadFile(configFilenameFlag)
	if err != nil {
		slog.Error("reading configuration file", "err", err)
		return
	}

	var cfg Configuration
	if err := toml.Unmarshal(configData, &cfg); err != nil {
		slog.Error("unmarshaling toml configuration", "err", err)
		return
	}

	cfg.Prefix = os.ExpandEnv(cfg.Prefix)
	for idx, item := range cfg.Secrets {
		cfg.Secrets[idx] = os.ExpandEnv(item)
	}

	ctx := context.Background()
	awscfg, err := config.LoadDefaultConfig(ctx, config.WithDefaultsMode(aws.DefaultsModeInRegion))
	if err != nil {
		slog.Error("error loading AWS configuration", "err", err)
		return
	}

	env := os.Environ()
	client := ssm.NewFromConfig(awscfg)

	nextToken := ""
	for {
		data, err := client.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
			Path:           &cfg.Prefix,
			NextToken:      aws.String(nextToken),
			Recursive:      aws.Bool(true),
			WithDecryption: aws.Bool(true),
		})
		if err != nil {
			slog.Error("error fetching parameters by path", "err", err)
			break
		}

		for _, param := range data.Parameters {
			if param.Value == nil {
				continue
			}
			name := parameterToEnv(param.Name)
			env = append(env, fmt.Sprintf("%s=%s", name, *param.Value))
		}

		if data.NextToken != nil && *data.NextToken != "" {
			nextToken = *data.NextToken
			continue
		}
		break
	}

	for _, chunk := range chunked(cfg.Secrets) {
		data, err := client.GetParameters(ctx, &ssm.GetParametersInput{
			Names:          chunk,
			WithDecryption: aws.Bool(true),
		})
		if err != nil {
			slog.Error("error fetching parameters", "err", err)
			continue
		}
		for _, param := range data.Parameters {
			if param.Value == nil {
				continue
			}
			name := parameterToEnv(param.Name)
			env = append(env, fmt.Sprintf("%s=%s", name, *param.Value))
		}
	}

	if err := syscall.Exec(argv[0], argv[1:], env); err != nil {
		slog.Error("error executing command", "cmd", argv, "err", err)
		os.Exit(1)
	}
}

var parameterNameRx *regexp.Regexp = regexp.MustCompile("/([^/]+)$")

func parameterToEnv(name *string) string {
	if name == nil {
		return ""
	}
	if m := parameterNameRx.FindStringSubmatch(*name); len(m) > 1 {
		return m[1]
	}
	return ""
}

func chunked(slice []string) [][]string {
	chunkSize := 10
	var chunks [][]string
	for len(slice) != 0 {
		if len(slice) < chunkSize {
			chunkSize = len(slice)
		}
		c := make([]string, chunkSize)
		copy(c, slice[:chunkSize])
		chunks = append(chunks, c)
		slice = slice[chunkSize:]
	}
	return chunks
}
