package server

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-seatbelt/seatbelt/internal/config"
	"github.com/go-seatbelt/seatbelt/internal/trace"
	"github.com/sirupsen/logrus"
)

var isDevelopment = os.Getenv("SEATBELT_ENV") != "production"

type server struct {
	srv  *http.Server
	done chan bool
	quit chan os.Signal
	cmd  *npmCmd
}

// Start starts our HTTP server, registering all routes on the given router.
//
// It will also serve any static files from /public.
func Start(fn func(r *chi.Mux)) {
	server := &server{}

	server.initialize()

	r := chi.NewMux()
	fn(r)
	server.registerRoutes(r)

	go server.listenForShutdown()

	server.start()
}

func (s *server) initialize() {
	s.srv = &http.Server{
		Addr: config.HTTPAddr,
	}

	s.done = make(chan bool, 1)
	s.quit = make(chan os.Signal, 1)
}

func (s *server) registerRoutes(r *chi.Mux) {
	publicFilepath := filepath.Join(config.RootPath, "public")

	// Serve assets from the /public directory.
	fileserver(r, "/public", http.Dir(publicFilepath))

	// Serve static HTML files within the views folder.
	r.NotFound(templateResolver)

	// Register the handler with our server, wrapping it with the Turbolinks
	// wrapper before doing so.
	s.srv.Handler = Turbolinks(r)
}

func (s *server) listenForShutdown() {
	// We assume that we're running in a goroutine, so we block until we
	// receive a quit signal to stop.
	// the server.
	<-s.quit
	logrus.Infoln("Server is shutting down.")

	if s.cmd.Cmd != nil {
		s.cmd.stop()
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	// Shutdown the server.
	s.srv.SetKeepAlivesEnabled(false)
	if err := s.srv.Shutdown(ctx); err != nil {
		logrus.Errorf("Could not gracefully shutdown the server: %+v", err)
	}

	// Inform the main goroutine that shutdown is complete.
	s.done <- true
}

func (s *server) start() {
	s.cmd = npmExec()

	signal.Notify(s.quit, os.Interrupt)

	logrus.WithFields(logrus.Fields{
		"database": config.Database,
		"username": config.Username,
		"password": config.Password,
	}).Info("Application started with configuration")

	logrus.Infof("Server is ready to handle requests at http://localhost%s", s.srv.Addr)
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.Fatalf("Could not listen on %s: %+v", s.srv.Addr, err)
	}

	// Block until the server shutdown process has completed.
	<-s.done
	logrus.Infoln("Server stopped")
}

// fileserver sets up a http.FileServer handler to serve static files from an
// http.FileSystem.
func fileserver(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		logrus.Fatalf("%s: fileserver does not permit URL parameters", trace.Getfl())
	}

	fs := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	}))
}

// templateResolver renders the template matching the current path, if it
// exists. If not, it returns 404 Not Found.
func templateResolver(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/")

	switch name {
	case "":
		// Render any "index.html" template that is present.
		f, err := os.Open(filepath.Join(config.RootPath, "views", "index.html"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, f); err != nil {
			logrus.Errorf("Error writing index.html file: %+v", err)
		}

	case "layout", "layout.html":
		http.Error(w, "The name layout is reserved", http.StatusBadRequest)
		return

	default:
		// Try to render a template matching the path.
		f, err := os.Open(filepath.Join(config.RootPath, "views", filepath.Clean(name)+".html"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, f); err != nil {
			logrus.Errorf("Error writing %s file: %+v", name, err)
		}
	}
}
