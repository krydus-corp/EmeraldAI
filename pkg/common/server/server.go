/*
 * File: server.go
 * Project: server
 * File Created: Friday, 17th June 2022 4:17:54 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package server

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/server/middleware/secure"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/acme/autocert"
)

// Basic Server
type Server struct {
	*echo.Echo
	cfg Config
}

// Config represents server specific config
type Config struct {
	Port              string
	TimeoutSeconds    int
	Domain            string
	CertCacheS3Bucket string
	KeyFile           string
	CrtFile           string
	Debug             bool
}

func New(cfg Config) *Server {
	e := echo.New()

	e.Logger.SetLevel(log.INFO)

	e.Pre(middleware.HTTPSWWWRedirect())
	e.Pre(middleware.NonWWWRedirect())
	e.Use(
		middleware.Logger(),
		middleware.Recover(),
		middleware.RateLimiter(middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate: 50, Burst: 200, ExpiresIn: 5 * time.Minute,
		})),
		secure.CORS(),
		secure.Headers(),
	)
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, "OK")
	})

	e.Validator = &CustomValidator{V: validator.New()}
	custErr := &customErrHandler{e: e}
	e.HTTPErrorHandler = custErr.handler
	e.Binder = &CustomBinder{b: &echo.DefaultBinder{}}

	return &Server{
		Echo: e,
		cfg:  cfg,
	}
}

// Start starts echo server
func (s Server) Start() {
	s.Echo.Debug = s.cfg.Debug

	s.Echo.Logger.Infof("Starting cert manager; caching certs to s3://certs/%s", s.cfg.CertCacheS3Bucket)
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(s.cfg.Domain, "localhost"),
		Cache:      S3Cache{Bucket: s.cfg.CertCacheS3Bucket, Prefix: "certs"},
	}

	tlsConfig := certManager.TLSConfig()
	tlsConfig.GetCertificate = getLocalOrLetsEncryptCert(s.Echo, &certManager, s.cfg.CrtFile, s.cfg.KeyFile)
	tlsConfig.InsecureSkipVerify = true

	timeout := time.Duration(s.cfg.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = time.Duration(2) * time.Hour
	}

	server := &http.Server{
		Addr:         s.cfg.Port,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
		IdleTimeout:  timeout,
		TLSConfig:    tlsConfig,
	}

	// Start server
	errChan := make(chan error)

	go func() {
		if err := s.Echo.StartServer(server); err != nil {
			errChan <- err
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 10 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		s.Echo.Logger.Infof("http.Server error; shutting down the server err=%s", err.Error())
	case <-quit:
		ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
		defer cancel()
		if err := s.Echo.Shutdown(ctx); err != nil {
			s.Echo.Logger.Fatal(err)
		}
	}
}

func getLocalOrLetsEncryptCert(e *echo.Echo, certManager *autocert.Manager, crtFile, keyFile string) func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {

		certificate, err := tls.LoadX509KeyPair(crtFile, keyFile)
		if err != nil {
			e.Logger.Infof("Falling back to Letsencrypt; err=%s", err)
			return certManager.GetCertificate(hello)
		}
		e.Logger.Infof("Loaded selfsigned certificate.")
		return &certificate, err
	}
}
