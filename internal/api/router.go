package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/medpath1024/OpenPrintHub/internal/print"
	"github.com/medpath1024/OpenPrintHub/internal/printer"
	"github.com/medpath1024/OpenPrintHub/internal/web"
)

// Config holds the server configuration
type Config struct {
	Port         int
	AllowOrigins string
	PrinterSvc   printer.Service
	PrintQueue   *print.Queue
}

// Server represents the API server
type Server struct {
	config   Config
	router   *gin.Engine
	handlers *Handlers
	wsHub    *WebSocketHub
}

// NewServer creates a new API server
func NewServer(config Config) *Server {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Create handlers
	handlers := NewHandlers(config.PrinterSvc, config.PrintQueue)

	// Create WebSocket hub
	wsHub := NewWebSocketHub()

	server := &Server{
		config:   config,
		router:   router,
		handlers: handlers,
		wsHub:    wsHub,
	}

	server.setupRoutes()
	server.setupStatusCallback()

	return server
}

// setupRoutes configures all routes
func (s *Server) setupRoutes() {
	// Apply middleware
	s.router.Use(RecoveryMiddleware())
	s.router.Use(LoggerMiddleware())
	s.router.Use(CORSMiddleware(s.config.AllowOrigins))
	s.router.Use(SecurityHeadersMiddleware())

	// Health check
	s.router.GET("/health", s.handlers.HealthCheck)

	// API v1 routes
	v1 := s.router.Group("/v1")
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

	// Web admin interface
	s.setupWebRoutes()
}

// setupWebRoutes configures the web admin interface routes
func (s *Server) setupWebRoutes() {
	// Create web handlers
	webHandlers := web.NewHandlers(s.config.PrinterSvc, s.config.PrintQueue)

	// Serve static files
	s.router.GET("/static/*filepath", func(c *gin.Context) {
		web.ServeStatic(c.Writer, c.Request)
	})

	// Web pages
	s.router.GET("/", webHandlers.Index)
	s.router.GET("/printers", webHandlers.Printers)
	s.router.GET("/jobs", webHandlers.Jobs)

	// HTMX partials
	s.router.GET("/partials/printers", webHandlers.PrintersPartial)
	s.router.GET("/partials/jobs", webHandlers.JobsPartial)
	s.router.GET("/partials/stats", webHandlers.StatsPartial)
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

// Run starts the server
func (s *Server) Run() error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	return http.ListenAndServe(addr, s.router)
}

// Router returns the underlying gin router (for testing)
func (s *Server) Router() *gin.Engine {
	return s.router
}
