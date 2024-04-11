package app

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/intob/logd/store"
	"golang.org/x/time/rate"
)

type App struct {
	logStore                 *store.Store
	rateLimitEvery           time.Duration
	rateLimitBurst           int
	laddrPort                string
	tlsCertFname             string
	tlsKeyFname              string
	accessControlAllowOrigin string
	commit                   string
	started                  time.Time
	clientMu                 sync.Mutex
	clients                  map[string]*client
	statusJson               []byte
}

type Cfg struct {
	LogStore                 *store.Store
	Commit                   []byte
	LaddrPort                string        `yaml:"laddr_port"`
	RateLimitEvery           time.Duration `yaml:"rate_limit_every"`
	RateLimitBurst           int           `yaml:"rate_limit_burst"`
	TLSCertFname             string        `yaml:"tls_cert_fname"`
	TLSKeyFname              string        `yaml:"tls_key_fname"`
	AccessControlAllowOrigin string        `yaml:"access_control_allow_origin"`
}

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewApp(ctx context.Context, cfg *Cfg) *App {
	app := &App{
		logStore:                 cfg.LogStore,
		rateLimitEvery:           cfg.RateLimitEvery,
		rateLimitBurst:           cfg.RateLimitBurst,
		laddrPort:                cfg.LaddrPort,
		tlsCertFname:             cfg.TLSCertFname,
		tlsKeyFname:              cfg.TLSKeyFname,
		accessControlAllowOrigin: cfg.AccessControlAllowOrigin,
		started:                  time.Now(),
		commit:                   string(cfg.Commit),
		clients:                  make(map[string]*client),
	}
	go app.cleanupClients()
	go app.measureStatus()
	go app.serve(ctx)
	return app
}

func (app *App) serve(ctx context.Context) {
	mux := http.NewServeMux()
	mux.Handle("/", app.rateLimitMiddleware(
		app.corsMiddleware(
			http.HandlerFunc(app.handleRequest))))
	server := &http.Server{Addr: app.laddrPort, Handler: mux}
	go func() {
		if app.tlsCertFname != "" {
			fmt.Println("app listening https on", app.laddrPort)
			err := server.ListenAndServeTLS(app.tlsCertFname, app.tlsKeyFname)
			if err != nil && err != http.ErrServerClosed {
				panic(fmt.Sprintf("failed to listen https: %v\n", err))
			}
		}
		fmt.Println("listening http on", app.laddrPort)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			panic(fmt.Sprintf("failed to listen http: %v\n", err))
		}
	}()
	<-ctx.Done()
	app.shutdown(server)
}

func (app *App) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	app.handleStatus(w)
}

func (app *App) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", app.accessControlAllowOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		next.ServeHTTP(w, r)
	})
}

// rateLimitMiddleware is a middleware that limits the rate of requests.
func (app *App) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := app.getRateLimiter(r)
		if !limiter.Allow() {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// getRateLimiter returns a rate limiter for the given request.
func (app *App) getRateLimiter(r *http.Request) *rate.Limiter {
	app.clientMu.Lock()
	defer app.clientMu.Unlock()
	key := r.Method + r.RemoteAddr
	v, exists := app.clients[key]
	if !exists {
		limiter := rate.NewLimiter(rate.Every(app.rateLimitEvery), app.rateLimitBurst)
		app.clients[key] = &client{limiter, time.Now()}
		return limiter
	}
	v.lastSeen = time.Now()
	return v.limiter
}

// cleanupClients removes clients that have not been seen for 10 seconds.
func (a *App) cleanupClients() {
	for {
		<-time.After(10 * time.Second)
		a.clientMu.Lock()
		for key, client := range a.clients {
			if time.Since(client.lastSeen) > 10*time.Second {
				delete(a.clients, key)
			}
		}
		a.clientMu.Unlock()
	}
}

// shutdown attempts to gracefully shutdown the server.
func (a *App) shutdown(server *http.Server) {
	// Create a context with timeout for the server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Attempt to gracefully shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		panic(fmt.Sprintf("server shutdown failed: %v", err))
	}
	fmt.Println("server shutdown gracefully")
}
