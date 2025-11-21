package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"melodee/internal/config"
	"melodee/internal/health"
	"melodee/internal/utils"
)

// Version of the application
var Version = "1.0.0"

func joinSlice(slice []string, separator string) string {
	if len(slice) == 0 {
		return ""
	}
	result := slice[0]
	for i := 1; i < len(slice); i++ {
		result += separator + slice[i]
	}
	return result
}

func main() {
	// Create configuration loader and load configuration
	configLoader := config.NewConfigLoader()
	appConfig, err := configLoader.Load()
	if err != nil {
		log.Fatal("Failed to load configuration: ", err)
	}

	// Verify FFmpeg and external tokens before starting the server
	if err := utils.ValidateFFmpegAndTokens(appConfig.Processing.Conversion.FFmpegPath); err != nil {
		log.Printf("Failed to validate prerequisites: %v", err)
		// For now we'll log the error but continue startup to allow configuration
		// In production, you might want to fail fast depending on requirements
	} else {
		log.Println("FFmpeg and external tokens validation passed")
	}

	// Create the Fiber app
	app := fiber.New(fiber.Config{
		ServerHeader: "Melodee",
		AppName:      "Melodee v" + Version,
	})

	// Setup middleware
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(helmet.New())
	corsConfig := cors.Config{
		AllowOrigins:     joinSlice(appConfig.Server.CORS.AllowOrigins, ","),
		AllowMethods:     joinSlice(appConfig.Server.CORS.AllowMethods, ","),
		AllowHeaders:     joinSlice(appConfig.Server.CORS.AllowHeaders, ","),
		AllowCredentials: appConfig.Server.CORS.AllowCredentials,
	}
	app.Use(cors.New(corsConfig))

	// Register health check endpoint
	health.RegisterHealthRoutes(app, appConfig)

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Shutting down gracefully...")
		if err := app.Shutdown(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
	}()

	// Start the server
	log.Printf("Starting server on %s:%d", appConfig.Server.Host, appConfig.Server.Port)
	if err := app.Listen(fmt.Sprintf("%s:%d", appConfig.Server.Host, appConfig.Server.Port)); err != nil {
		log.Printf("Error starting server: %v", err)
	}
}