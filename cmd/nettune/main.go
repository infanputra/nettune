// Package main provides the CLI entry point for nettune
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jtsang4/nettune/internal/client/mcp"
	"github.com/jtsang4/nettune/internal/server/api"
	"github.com/jtsang4/nettune/internal/shared/config"
	"github.com/jtsang4/nettune/pkg/version"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	rootCmd = &cobra.Command{
		Use:   "nettune",
		Short: "Network diagnostics and TCP optimization tool",
		Long: `Nettune is a network diagnostics and TCP optimization tool that provides:
- End-to-end network testing (RTT, throughput, latency under load)
- Configuration profiles for network optimization (BBR, FQ, buffer tuning)
- Safe apply/rollback mechanism with snapshots
- MCP integration for AI-assisted optimization`,
	}

	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Start nettune in server mode",
		Long:  `Start the nettune HTTP API server that handles probe requests and system configuration.`,
		RunE:  runServer,
	}

	clientCmd = &cobra.Command{
		Use:   "client",
		Short: "Start nettune in client mode (MCP stdio server)",
		Long:  `Start the nettune MCP stdio server for integration with Chat GUI.`,
		RunE:  runClient,
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			info := version.GetInfo()
			fmt.Println(info.String())
		},
	}

	// Server flags
	serverAPIKey       string
	serverListen       string
	serverStateDir     string
	serverReadTimeout  int
	serverWriteTimeout int

	// Client flags
	clientAPIKey  string
	clientServer  string
	clientTimeout int
)

func init() {
	// Server flags
	serverCmd.Flags().StringVar(&serverAPIKey, "api-key", "", "API key for authentication (required)")
	serverCmd.Flags().StringVar(&serverListen, "listen", "0.0.0.0:9876", "Address to listen on")
	serverCmd.Flags().StringVar(&serverStateDir, "state-dir", "", "Directory for state storage")
	serverCmd.Flags().IntVar(&serverReadTimeout, "read-timeout", 30, "HTTP read timeout in seconds")
	serverCmd.Flags().IntVar(&serverWriteTimeout, "write-timeout", 60, "HTTP write timeout in seconds")
	serverCmd.MarkFlagRequired("api-key")

	// Client flags
	clientCmd.Flags().StringVar(&clientAPIKey, "api-key", "", "API key for authentication (required)")
	clientCmd.Flags().StringVar(&clientServer, "server", "http://127.0.0.1:9876", "Server URL")
	clientCmd.Flags().IntVar(&clientTimeout, "timeout", 60, "Request timeout in seconds")
	clientCmd.MarkFlagRequired("api-key")

	// Add commands
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(clientCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runServer(cmd *cobra.Command, args []string) error {
	// Create logger (output to stderr to not interfere with stdout)
	logger := createLogger(false)
	defer logger.Sync()

	// Build config
	cfg := config.DefaultServerConfig()
	cfg.APIKey = serverAPIKey
	cfg.Listen = serverListen
	cfg.ReadTimeout = serverReadTimeout
	cfg.WriteTimeout = serverWriteTimeout

	if serverStateDir != "" {
		cfg.StateDir = serverStateDir
	}

	logger.Info("starting nettune server",
		zap.String("version", version.Version),
		zap.String("listen", cfg.Listen),
		zap.String("state_dir", cfg.StateDir))

	// Create and start server
	server, err := api.NewServer(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		logger.Info("shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			logger.Error("error during shutdown", zap.Error(err))
		}
	}()

	return server.Start()
}

func runClient(cmd *cobra.Command, args []string) error {
	// Create logger (output to stderr, MCP uses stdout)
	logger := createLogger(true)
	defer logger.Sync()

	logger.Info("starting nettune client (MCP mode)",
		zap.String("version", version.Version),
		zap.String("server", clientServer))

	// Create MCP server
	mcpServer := mcp.NewServer(
		clientServer,
		clientAPIKey,
		time.Duration(clientTimeout)*time.Second,
		logger,
	)

	// Start MCP server (blocks until stdin closes)
	return mcpServer.Start()
}

func createLogger(quiet bool) *zap.Logger {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	level := zap.InfoLevel
	if quiet {
		level = zap.WarnLevel
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stderr), // Always log to stderr
		level,
	)

	return zap.New(core)
}
