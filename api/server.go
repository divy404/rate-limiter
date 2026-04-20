package api

import (
	"github.com/divy404/rate-limiter/internal/store"
	"github.com/gin-gonic/gin"
)

type Server struct {
	router     *gin.Engine
	redisStore *store.RedisStore
}

func NewServer(redisStore *store.RedisStore) *Server {
	s := &Server{
		router:     gin.Default(),
		redisStore: redisStore,
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	// v1 := s.router.Group("/api/v1")
	{
		// v1.POST("/check", s.handleCheck)
		// v1.GET("/status", s.handleStatus)
	}
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}