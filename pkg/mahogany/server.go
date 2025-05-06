package mahogany

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path"
)

type Server struct {
	config       Config
	view         *ViewFinder
	httpServer   *http.Server
	updateServer *UpdateServer
}

func NewServer(config Config, updateServer *UpdateServer) (*Server, error) {
	mux := http.NewServeMux()
	viewFinder, err := NewViewFinder(config)
	if err != nil {
		return nil, err
	}

	s := &Server{
		config: config,
		view:   viewFinder,
		httpServer: &http.Server{
			Addr:         fmt.Sprintf("0.0.0.0:%d", config.Port),
			ReadTimeout:  config.Timeout,
			WriteTimeout: config.Timeout,
			Handler:      mux,
		},
		updateServer: updateServer,
	}

	mux.HandleFunc("GET /{$}", s.HandleIndex)
	mux.HandleFunc("GET /container/{containerID}", s.HandleContainer)
	mux.HandleFunc("GET /container/{containerID}/inspect", s.HandleContainerInspect)
	mux.HandleFunc("POST /container/{containerID}/start", s.HandleContainerStart)
	mux.HandleFunc("POST /container/{containerID}/stop", s.HandleContainerStop)
	mux.HandleFunc("POST /container/{containerID}/restart", s.HandleContainerRestart)
	mux.HandleFunc("DELETE /container/{containerID}/delete", s.HandleContainerDelete)
	mux.HandleFunc("GET /container/{containerID}/logs", s.HandleContainerLogs)
	mux.HandleFunc("GET /container/{containerID}/logs/stream", s.HandleContainerLogsStream)
	mux.HandleFunc("GET /registry", s.HandleRegistry)
	mux.HandleFunc("DELETE /registry/image/{repository}/{digest}", s.HandleRegistryImageDelete)
	mux.HandleFunc("GET /watchtower", s.HandleWatchtower)
	mux.HandleFunc("POST /watchtower/update", s.HandleWatchtowerUpdate)
	mux.HandleFunc("POST /github/webhook", s.HandleGithubWebHook)
	mux.Handle("GET /static/", http.StripPrefix("/static", http.FileServer(http.Dir(config.StaticDir))))

	slog.Info("loaded mux", "routes", mux)
	return s, nil
}

func (s *Server) Start() error {
	slog.Info("starting server", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()
	s.httpServer.Shutdown(ctx)
}

func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	plate, err := loadTemplates(s.config.StaticDir)
	if err != nil {
		slog.Error("failed to load templates", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	view, err := s.view.GetIndex(r.Context())
	if err != nil {
		slog.Error("failed to get index view", "err", err)
	}
	if err = plate.ExecuteTemplate(w, "IndexView", view); err != nil {
		slog.Error("failed to execute index template", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) HandleContainer(w http.ResponseWriter, r *http.Request) {
	plate, err := loadTemplates(s.config.StaticDir)
	if err != nil {
		slog.Error("failed to load templates", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	view, err := s.view.GetContainer(r.Context(), r.PathValue("containerID"))
	if err != nil {
		slog.Error("failed to get container view", "err", err)
	}
	if err = plate.ExecuteTemplate(w, "ContainerView", view); err != nil {
		slog.Error("failed to execute container template", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) HandleRegistry(w http.ResponseWriter, r *http.Request) {
	plate, err := loadTemplates(s.config.StaticDir)
	if err != nil {
		slog.Error("failed to load templates", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	view, err := s.view.GetRegistry(r.Context())
	if err != nil {
		slog.Error("failed to get registry view", "err", err)
	}
	if err = plate.ExecuteTemplate(w, "RegistryView", view); err != nil {
		slog.Error("failed to execute registry template", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) HandleWatchtower(w http.ResponseWriter, r *http.Request) {
	plate, err := loadTemplates(s.config.StaticDir)
	if err != nil {
		slog.Error("failed to load templates", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	view, err := s.view.GetWatchtower(r.Context())
	if err != nil {
		slog.Error("failed to get watchtower view", "err", err)
	}
	if err = plate.ExecuteTemplate(w, "WatchtowerView", view); err != nil {
		slog.Error("failed to execute watchtower template", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) HandleContainerInspect(w http.ResponseWriter, r *http.Request) {
	plate, err := loadTemplates(s.config.StaticDir)
	if err != nil {
		slog.Error("failed to load templates", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	view, err := s.view.GetContainer(r.Context(), r.PathValue("containerID"))
	if err != nil {
		slog.Error("failed to get container view", "err", err)
	}
	if err = plate.ExecuteTemplate(w, "container", view); err != nil {
		slog.Error("failed to execute container template", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) HandleContainerStart(w http.ResponseWriter, r *http.Request) {
	plate, err := loadTemplates(s.config.StaticDir)
	if err != nil {
		slog.Error("failed to load templates", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	view, _ := s.view.StartContainer(r.Context(), r.PathValue("containerID"))
	if err = plate.ExecuteTemplate(w, "container-start", view); err != nil {
		slog.Error("failed to execute container start template", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) HandleContainerStop(w http.ResponseWriter, r *http.Request) {
	plate, err := loadTemplates(s.config.StaticDir)
	if err != nil {
		slog.Error("failed to load templates", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	view, _ := s.view.StopContainer(r.Context(), r.PathValue("containerID"))
	if err = plate.ExecuteTemplate(w, "container-stop", view); err != nil {
		slog.Error("failed to execute container stop template", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) HandleContainerRestart(w http.ResponseWriter, r *http.Request) {
	plate, err := loadTemplates(s.config.StaticDir)
	if err != nil {
		slog.Error("failed to load templates", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	view, _ := s.view.RestartContainer(r.Context(), r.PathValue("containerID"))
	if err = plate.ExecuteTemplate(w, "container-restart", view); err != nil {
		slog.Error("failed to execute container restart template", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) HandleContainerDelete(w http.ResponseWriter, r *http.Request) {
	plate, err := loadTemplates(s.config.StaticDir)
	if err != nil {
		slog.Error("failed to load templates", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	view, _ := s.view.RemoveContainer(r.Context(), r.PathValue("containerID"))
	if err = plate.ExecuteTemplate(w, "container-delete", view); err != nil {
		slog.Error("failed to execute container delete template", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) HandleContainerLogs(w http.ResponseWriter, r *http.Request) {
	plate, err := loadTemplates(s.config.StaticDir)
	if err != nil {
		slog.Error("failed to load templates", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	view, _ := s.view.GetContainer(r.Context(), r.PathValue("containerID"))
	if err = plate.ExecuteTemplate(w, "container-logs", view); err != nil {
		slog.Error("failed to execute container logs template", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) HandleContainerLogsStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/event-stream")
	w.Header().Set("cache-control", "no-cache")
	w.Header().Set("connection", "keep-alive")

	logs, err := s.view.GetContainerLogs(r.Context(), r.PathValue("containerID"))
	if err != nil {
		fmt.Fprint(w, "event: log\n")
		fmt.Fprintf(w, "data: Error: %v\n\n", err)
		return
	}
	defer logs.Close()
	buffer := bufio.NewScanner(logs)
	for buffer.Scan() {
		fmt.Fprint(w, "event: log\n")
		fmt.Fprintf(w, "data: <p>%s</p>\n\n", buffer.Bytes())
	}
}

func (s *Server) HandleRegistryImageDelete(w http.ResponseWriter, r *http.Request) {
	plate, err := loadTemplates(s.config.StaticDir)
	if err != nil {
		slog.Error("failed to load templates", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	view, _ := s.view.DeleteRegistryImage(r.Context(), r.PathValue("repository"), r.PathValue("digest"))
	if err = plate.ExecuteTemplate(w, "toast", view); err != nil {
		slog.Error("failed to execute toast template", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) HandleWatchtowerUpdate(w http.ResponseWriter, r *http.Request) {
	plate, err := loadTemplates(s.config.StaticDir)
	if err != nil {
		slog.Error("failed to load templates", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	view := s.view.WatchtowerUpdate(r.Context())
	if err = plate.ExecuteTemplate(w, "toast", view); err != nil {
		slog.Error("failed to execute toast template", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) HandleGithubWebHook(w http.ResponseWriter, r *http.Request) {
	var event GithubReleaseEvent
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&event); err != nil {
		slog.Error("failed to decode github webhook", "err", err)
		return
	}
	// TODO check github signature
	slog.Info("received github webhook", "name", *event.Repo.Name)
	s.updateServer.PropagateGithubRelease(&event)
}

func loadTemplates(baseDir string) (plate *template.Template, err error) {
	plate = template.New("").Funcs(template.FuncMap{
		"truncate": func(str string, maxLen int) string {
			maxLen = min(len(str), maxLen)
			return str[0:maxLen]
		},
	})
	plate, err = plate.ParseGlob(path.Join(baseDir, "views/*.html"))
	if err != nil {
		return
	}
	plate, err = plate.ParseGlob(path.Join(baseDir, "templates/*.html"))
	return
}
