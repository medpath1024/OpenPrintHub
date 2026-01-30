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
	version     = "0.1.0"
)

func main() {
	// Command line flags
	port := flag.Int("port", defaultPort, "HTTP server port")
	showVersion := flag.Bool("version", false, "Show version")
	allowOrigins := flag.String("cors", "*", "Comma-separated list of allowed CORS origins")
	flag.Parse()

	if *showVersion {
		fmt.Printf("OpenPrintHub v%s\n", version)
		os.Exit(0)
	}

	// Initialize printer service
	printerSvc := printer.New()

	// Initialize print queue
	printQueue := print.NewQueue(printerSvc)

	// Initialize and start API server
	server := api.NewServer(api.Config{
		Port:         *port,
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

	log.Printf("OpenPrintHub v%s starting on http://localhost:%d\n", version, *port)
	if err := server.Run(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
