package cmd

/*
Copyright © 2024 Ganeshdip Dumbare <ganeshdip.dumbare@gmail.com>
*/

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"deblock/config"
	"deblock/internal/address"
	"deblock/internal/api/rest"
	"deblock/internal/blockchain"
	"deblock/internal/dlock"
	"deblock/internal/pubsub"
	"deblock/internal/txmonitor"

	"github.com/spf13/cobra"
)

// @title Deblock Transaction Monitor API
// @version 1.0
// @description This is a Deblock transaction monitor service API
// @termsOfService http://swagger.io/terms/

// @contact.name Ganeshdip Dumbare
// @contact.url https://github.com/ganeshdipdumbare
// @contact.email ganeshdip.dumbare@gmail.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// restCmd represents the rest command
var restCmd = &cobra.Command{
	Use:   "rest",
	Short: "Start the REST API server",
	Long: `This command initializes and starts the REST API server.
It sets up the necessary routes and listens for incoming HTTP requests.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create logger instance first for early logging
		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug, // Start with debug to capture all logs
		}))

		// Log start of configuration loading
		logger.Info("Starting Deblock Transaction Monitor",
			"version", "1.0",
			"command", "rest",
		)

		// Load the configuration with detailed logging
		config, err := config.LoadConfig()
		if err != nil {
			logger.Error("Failed to load configuration",
				"error", err,
				"error_type", fmt.Sprintf("%T", err),
			)

			// Provide more context about potential configuration issues
			switch {
			case config == nil:
				logger.Error("Configuration is nil. Check environment variables and config files.")
			case config.EthereumRPCURL == "":
				logger.Warn("Ethereum RPC URL is not set. This may cause issues with blockchain monitoring.")
			case config.RedisURL == "":
				logger.Error("Redis URL is not set. Distributed locking will not work.")
			case config.KafkaBrokers == nil || len(config.KafkaBrokers) == 0:
				logger.Error("Kafka brokers are not configured. Event publishing will fail.")
			}

			os.Exit(1)
		}

		// Log loaded configuration details (be careful not to log sensitive information)
		fmt.Printf("Configuration loaded successfully: %+v\n", config)

		// Create blockchain client
		blockchainClient, err := blockchain.NewEthereumClient(
			logger,
			config.EthereumRPCURL,
			config.EthereumWSURL,
		)
		if err != nil {
			logger.Error("Failed to create blockchain client",
				"error", err,
				"rpc_url", config.EthereumRPCURL,
			)
			os.Exit(1)
		}

		// Create address watcher
		addressWatcher := address.NewInMemoryAddressWatcher()

		// Add watched addresses to address watcher
		if len(config.WatchedAddresses) > 0 {
			logger.Info("Adding watched addresses",
				"count", len(config.WatchedAddresses),
			)
			addressWatcher.AddAddresses(cmd.Context(), config.WatchedAddresses)
		}

		// Create distributed lock
		var redisAddr string
		if strings.HasPrefix(config.RedisURL, "redis://") {
			// Remove redis:// prefix
			redisAddr = strings.TrimPrefix(config.RedisURL, "redis://")
		} else {
			redisAddr = config.RedisURL
		}
		distributedLock := dlock.NewRedsyncLock(redisAddr)

		// Create publisher
		publisher, err := pubsub.NewKafkaWatermillPublisher(logger, config.KafkaBrokers)
		if err != nil {
			logger.Error("Failed to create publisher",
				"error", err,
				"kafka_brokers", config.KafkaBrokers,
			)
			os.Exit(1)
		}

		// Create transaction monitor service
		txMonitorService := txmonitor.NewTxMonitorService(
			logger,
			blockchainClient,
			addressWatcher,
			publisher,
			distributedLock,
		)

		// Create a new rest api instance
		api, err := rest.NewApi(logger, config.ServerPort, txMonitorService)
		if err != nil {
			logger.Error("Failed to create new rest api",
				"error", err,
				"server_port", config.ServerPort,
			)
			os.Exit(1)
		}

		// Start the rest server
		api.StartServer()
	},
}

func init() {
	rootCmd.AddCommand(restCmd)
}
