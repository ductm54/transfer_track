// Package server provides HTTP server functionality.
package server

import (
	"fmt"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server to serve the service.
type Server struct {
	s        *gin.Engine
	bindAddr string
	l        *zap.SugaredLogger
}

// New returns a new server.
func New(bindAddr string) *Server {
	engine := gin.New()
	engine.Use(gin.Recovery())

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	engine.Use(cors.New(config))

	s := &Server{
		s:        engine,
		bindAddr: bindAddr,
		l:        zap.S(),
	}

	gin.SetMode(gin.ReleaseMode)
	s.register()

	return s
}

// Run runs server.
func (s *Server) Run() error {
	if err := s.s.Run(s.bindAddr); err != nil {
		return fmt.Errorf("run server: %w", err)
	}

	return nil
}

// GetEngine returns the underlying gin engine.
func (s *Server) GetEngine() *gin.Engine {
	return s.s
}

func (s *Server) register() {
	pprof.Register(s.s, "/debug")
}
