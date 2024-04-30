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
	"gopkg.in/yaml.v2"
)

type Configuration struct {
	Prefix string `yaml:"prefix"`
	Secrets []string `yaml:"secrets"`
}

var Version string
var log *slog.Logger

func main() {
	var (
		verbose, version, humanOutput bool
		configFilename string
	)
	flag.BoolVar(&version, "version", false, "show the current version")
	flag.BoolVar(&verbose, "v", false, "set verbose output")
	flag.BoolVar(&verbose, "verbose", false, "set verbose output")
	flag.BoolVar(&humanOutput, "H", false, "humanly readable logging")
	flag.StringVar(&configFilename, "config", "sesame.yaml", "config file path")
	flag.StringVar(&configFilename, "c", "sesame.yaml", "config file path")

	argv := flag.Args()

	if version {
		log.Info("sesame", "version", Version)
		os.Exit(0)
		return
	}

	var logHandler slog.Handler
	if humanOutput && verbose {
		logHandler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
			AddSource: true,
		})
	} else if humanOutput {
		logHandler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})
	} else if verbose {
		logHandler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
			AddSource: true,
		})
	} else {
		logHandler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})
	}
	log = slog.New(logHandler)

	configData, err := os.ReadFile(configFilename)
	if err != nil {
		slog.Error("error reading configuration file", "err", err)
		return
	}

	var cfg Configuration
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		slog.Error("error unmashaling yaml configuration", "err", err)
		return
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
			Path: &cfg.Prefix,
			NextToken: aws.String(nextToken),
			Recursive: aws.Bool(true),
		})
		if err != nil {
			log.Error("error fetching parameters by path", "err", err)
			continue
		}

		for _, param := range data.Parameters {
			name := parameterToEnv(param.Name)
			env = append(env, fmt.Sprintf("%s=%s", name, *param.Value))
		}

		if *data.NextToken != "" {
			nextToken = *data.NextToken
			continue
		}
		break
	}

	// Secrets, requested in batches of max 10 paths
	for _, chunk := range chunked(cfg.Secrets) {
		data, err := client.GetParameters(ctx, &ssm.GetParametersInput{
			Names: chunk,
		})
		if err != nil {
			log.Error("error fetching parameters", "err", err)
			continue
		}
		for _, param := range data.Parameters {
			name := parameterToEnv(param.Name)
			env = append(env, fmt.Sprintf("%s=%s", name, *param.Value))
		}
	}

	// exec(1) and give up control
	if err := syscall.Exec(argv[0], argv[1:], env); err != nil {
		slog.Error("error executing command", "cmd", argv, "err", err)
		os.Exit(1)
	}
}

var parameterNameRx *regexp.Regexp = regexp.MustCompile("/([^/]+)$")

func parameterToEnv(name *string) string {
	return parameterNameRx.FindString(*name)
}
func chunked(slice []string) [][]string {
	chunkSize := 10
	var chunks [][]string
	for {
		if len(slice) == 0 {
			break
		}

		// necessary check to avoid slicing beyond
		// slice capacity
		if len(slice) < chunkSize {
			chunkSize = len(slice)
		}

		chunks = append(chunks, slice[0:chunkSize])
		slice = slice[chunkSize:]
	}

	return chunks
}
