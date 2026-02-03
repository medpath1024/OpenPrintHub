package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/medpath1024/OpenPrintHub/internal/api"
	"github.com/medpath1024/OpenPrintHub/internal/print"
	"github.com/medpath1024/OpenPrintHub/internal/printer"
)

const (
	defaultPort = 16800
)

var (
	// version is injected at build time via -ldflags "-X main.version=..."
	version = "0.1.4"
)

func main() {
	// Command line flags
	port := flag.Int("port", defaultPort, "API server port")
	webPort := flag.Int("web-port", 0, "Web admin dashboard port (default: port+1)")
	showVersion := flag.Bool("version", false, "Show version")
	allowOrigins := flag.String("cors", "*", "Comma-separated list of allowed CORS origins")
	flag.Parse()

	if *showVersion {
		fmt.Printf("OpenPrintHub v%s\n", version)
		os.Exit(0)
	}

	// Set web port to port+1 if not explicitly specified
	if *webPort == 0 {
		*webPort = *port + 1
	}

	// Initialize printer service
	printerSvc := printer.New()

	// Initialize print queue
	printQueue := print.NewQueue(printerSvc)

	// Initialize and start API server
	server := api.NewServer(api.Config{
		Port:         *port,
		WebPort:      *webPort,
		Version:      version,
		AllowOrigins: *allowOrigins,
		PrinterSvc:   printerSvc,
		PrintQueue:   printQueue,
	})

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")
		printQueue.Stop()
		os.Exit(0)
	}()

	log.Printf("OpenPrintHub v%s starting...\n", version)
	log.Printf("  API server:    http://localhost:%d\n", *port)
	log.Printf("  Admin dashboard: http://localhost:%d\n", *webPort)
	if err := server.Run(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
