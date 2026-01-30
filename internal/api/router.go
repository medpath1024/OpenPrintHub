package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/medpath1024/OpenPrintHub/internal/print"
	"github.com/medpath1024/OpenPrintHub/internal/printer"
	"github.com/medpath1024/OpenPrintHub/internal/web"
)

// Config holds the server configuration
type Config struct {
	Port         int    // API/communication port
	WebPort      int    // Web admin dashboard port (typically Port+1)
	AllowOrigins string
	PrinterSvc   printer.Service
	PrintQueue   *print.Queue
}

// Server represents the API server
type Server struct {
	config    Config
	apiRouter *gin.Engine
	webRouter *gin.Engine
	handlers  *Handlers
	wsHub     *WebSocketHub
}

// NewServer creates a new API server
func NewServer(config Config) *Server {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	apiRouter := gin.New()
	webRouter := gin.New()

	// Create handlers
	handlers := NewHandlers(config.PrinterSvc, config.PrintQueue)

	// Create WebSocket hub
	wsHub := NewWebSocketHub()

	server := &Server{
		config:    config,
		apiRouter: apiRouter,
		webRouter: webRouter,
		handlers:  handlers,
		wsHub:     wsHub,
	}

	server.setupAPIRoutes()
	server.setupWebRoutes()
	server.setupStatusCallback()

	return server
}

// setupAPIRoutes configures API routes on the API router
func (s *Server) setupAPIRoutes() {
	// Apply middleware
	s.apiRouter.Use(RecoveryMiddleware())
	s.apiRouter.Use(LoggerMiddleware())
	s.apiRouter.Use(CORSMiddleware(s.config.AllowOrigins))
	s.apiRouter.Use(SecurityHeadersMiddleware())

	// Health check
	s.apiRouter.GET("/health", s.handlers.HealthCheck)

	// API v1 routes
	v1 := s.apiRouter.Group("/v1")
	{
		// Printers
		v1.GET("/printers", s.handlers.ListPrinters)
		v1.GET("/printers/default", s.handlers.GetDefaultPrinter)
		v1.GET("/printers/:id/status", s.handlers.GetPrinterStatus)

		// Print jobs
		v1.POST("/print", s.handlers.SubmitPrintJob)
		v1.GET("/jobs", s.handlers.ListJobs)
		v1.GET("/jobs/:id", s.handlers.GetJobStatus)

		// Stats
		v1.GET("/stats", s.handlers.GetStats)

		// WebSocket
		v1.GET("/ws", s.handleWebSocket)
	}
}

// setupWebRoutes configures the web admin interface routes
func (s *Server) setupWebRoutes() {
	// Apply middleware
	s.webRouter.Use(RecoveryMiddleware())
	s.webRouter.Use(LoggerMiddleware())
	s.webRouter.Use(SecurityHeadersMiddleware())

	// Create web handlers
	webHandlers := web.NewHandlers(s.config.PrinterSvc, s.config.PrintQueue)

	// Serve static files
	s.webRouter.GET("/static/*filepath", func(c *gin.Context) {
		web.ServeStatic(c.Writer, c.Request)
	})

	// Web pages
	s.webRouter.GET("/", webHandlers.Index)
	s.webRouter.GET("/printers", webHandlers.Printers)
	s.webRouter.GET("/jobs", webHandlers.Jobs)

	// HTMX partials
	s.webRouter.GET("/partials/printers", webHandlers.PrintersPartial)
	s.webRouter.GET("/partials/jobs", webHandlers.JobsPartial)
	s.webRouter.GET("/partials/stats", webHandlers.StatsPartial)
}

// setupStatusCallback registers the status callback for WebSocket updates
func (s *Server) setupStatusCallback() {
	s.config.PrintQueue.OnStatusChange(func(jobID string, status printer.JobStatus, message string) {
		s.wsHub.Broadcast(WebSocketMessage{
			Type: "job_status",
			Data: map[string]interface{}{
				"job_id":  jobID,
				"status":  status,
				"message": message,
			},
		})
	})

	// Start the WebSocket hub
	go s.wsHub.Run()
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(c *gin.Context) {
	s.wsHub.HandleConnection(c.Writer, c.Request)
}

// Run starts both API and Web servers
func (s *Server) Run() error {
	// Start Web server in a goroutine
	go func() {
		webAddr := fmt.Sprintf(":%d", s.config.WebPort)
		log.Printf("Web admin dashboard running on http://localhost:%d\n", s.config.WebPort)
		if err := http.ListenAndServe(webAddr, s.webRouter); err != nil {
			log.Printf("Web server error: %v", err)
		}
	}()

	// Start API server (blocks)
	apiAddr := fmt.Sprintf(":%d", s.config.Port)
	return http.ListenAndServe(apiAddr, s.apiRouter)
}

// APIRouter returns the API gin router (for testing)
func (s *Server) APIRouter() *gin.Engine {
	return s.apiRouter
}

// WebRouter returns the Web gin router (for testing)
func (s *Server) WebRouter() *gin.Engine {
	return s.webRouter
}
