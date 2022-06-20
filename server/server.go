package http

import (
	"context"
	"net/http"
	"time"

	"github.com/OdyseeTeam/e-mage/internal/metrics"

	"github.com/OdyseeTeam/gody-cdn/store"
	"github.com/OdyseeTeam/mirage/optimizer"
	"github.com/bluele/gcache"
	nice "github.com/ekyoung/gin-nice-recovery"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lbryio/lbry.go/v2/extras/stop"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/singleflight"
)

// Server is an instance of a peer server that houses the listener and store.
type Server struct {
	grp        *stop.Group
	optimizer  *optimizer.Optimizer
	cache      store.ObjectStore
	errorCache gcache.Cache
	sf         *singleflight.Group
}

// NewServer returns an initialized Server pointer.
func NewServer(optimizer *optimizer.Optimizer, cache store.ObjectStore) *Server {
	return &Server{
		grp:        stop.New(),
		optimizer:  optimizer,
		cache:      cache,
		errorCache: gcache.New(10000).Expiration(2 * time.Minute).Build(),
		sf:         &singleflight.Group{},
	}
}

// Shutdown gracefully shuts down the peer server.
func (s *Server) Shutdown() {
	log.Debug("shutting down HTTP server")
	s.grp.StopAndWait()
	log.Debug("HTTP server stopped")
}

// Start starts the server listener to handle connections.
func (s *Server) Start(address string) error {
	//gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(s.ErrorHandle)
	router.Use(nice.Recovery(s.recoveryHandler))
	router.Use(s.addCSPHeaders)
	router.Use(cors.Default())
	metrics.InstallRoute(router)
	router.GET("/r/:resource", s.getImageHandler)
	router.POST("/upload", s.uploadHandler)
	router.POST("/upload.php", s.uploadHandler)
	srv := &http.Server{
		Addr:    address,
		Handler: router,
	}
	go s.listenForShutdown(srv)
	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	s.grp.Add(1)
	go func() {
		defer s.grp.Done()
		log.Println("HTTP server listening on " + address)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	return nil
}

func (s *Server) listenForShutdown(listener *http.Server) {
	<-s.grp.Ch()
	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := listener.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
}
