package mahogany

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	db "github.com/mpoegel/mahogany/internal/db"
	views "github.com/mpoegel/mahogany/pkg/mahogany/views"
)

type Server struct {
	config       Config
	view         *views.ViewFinder
	httpServer   *http.Server
	updateServer *UpdateServer
}

func NewServer(config Config, updateServer *UpdateServer) (*Server, error) {
	mux := http.NewServeMux()
	viewFinder, err := views.NewViewFinder(config.DockerHost, config.DockerVersion, config.DbFile)
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

	mux.HandleFunc("GET /{$}", s.newHandler(func(r *http.Request) Viewer {
		return s.view.GetIndex(r.Context())
	}))
	mux.HandleFunc("GET /container/{containerID}", s.newHandler(func(r *http.Request) Viewer {
		return s.view.GetContainer(r.Context(), r.PathValue("containerID")).WithName("ContainerView")
	}))
	mux.HandleFunc("GET /container/{containerID}/inspect", s.newHandler(func(r *http.Request) Viewer {
		return s.view.GetContainer(r.Context(), r.PathValue("containerID")).WithName("container")
	}))
	mux.HandleFunc("POST /container/{containerID}/start", s.newHandler(func(r *http.Request) Viewer {
		return s.view.StartContainer(r.Context(), r.PathValue("containerID"))
	}))
	mux.HandleFunc("POST /container/{containerID}/stop", s.newHandler(func(r *http.Request) Viewer {
		return s.view.StopContainer(r.Context(), r.PathValue("containerID"))
	}))
	mux.HandleFunc("POST /container/{containerID}/restart", s.newHandler(func(r *http.Request) Viewer {
		return s.view.RestartContainer(r.Context(), r.PathValue("containerID"))
	}))
	mux.HandleFunc("DELETE /container/{containerID}/delete", s.newHandler(func(r *http.Request) Viewer {
		return s.view.RemoveContainer(r.Context(), r.PathValue("containerID"))
	}))
	mux.HandleFunc("GET /container/{containerID}/logs", s.newHandler(func(r *http.Request) Viewer {
		return s.view.GetContainer(r.Context(), r.PathValue("containerID")).WithName("container-logs")
	}))
	mux.HandleFunc("GET /container/{containerID}/logs/stream", s.HandleContainerLogsStream)
	mux.HandleFunc("GET /registry", s.newHandler(func(r *http.Request) Viewer {
		return s.view.GetRegistry(r.Context())
	}))
	mux.HandleFunc("DELETE /registry/image/{repository}/{digest}", s.newHandler(func(r *http.Request) Viewer {
		return s.view.DeleteRegistryImage(r.Context(), r.PathValue("repository"), r.PathValue("digest"))
	}))
	mux.HandleFunc("GET /watchtower", s.newHandler(func(r *http.Request) Viewer {
		return s.view.GetWatchtower(r.Context())
	}))
	mux.HandleFunc("POST /watchtower/update", s.newHandler(func(r *http.Request) Viewer {
		return s.view.WatchtowerUpdate(r.Context())
	}))
	mux.HandleFunc("POST /github/webhook", s.HandleGithubWebHook)
	mux.HandleFunc("GET /control-plane", s.newHandler(func(r *http.Request) Viewer {
		return s.view.GetControlPlane(r.Context())
	}))
	mux.HandleFunc("GET /settings", s.newHandler(func(r *http.Request) Viewer {
		return s.view.GetSettings(r.Context())
	}))
	mux.HandleFunc("POST /settings", s.HandlePostSettings)
	mux.HandleFunc("GET /devices", s.newHandler(func(r *http.Request) Viewer {
		return s.view.GetDevices(r.Context())
	}))
	mux.HandleFunc("GET /device/{deviceID}", s.newHandler(func(r *http.Request) Viewer {
		return s.view.GetDevice(r.Context(), r.PathValue("deviceID"))
	}))
	mux.HandleFunc("GET /packages", s.newHandler(func(r *http.Request) Viewer {
		return s.view.GetPackages(r.Context()).WithName("PackagesView")
	}))
	mux.HandleFunc("POST /package", s.newHandler(func(r *http.Request) Viewer {
		return s.view.AddPackage(r.Context(), db.AddPackageParams{
			Name:       r.FormValue("Name"),
			InstallCmd: r.FormValue("InstallCmd"),
			UpdateCmd:  r.FormValue("UpdateCmd"),
			RemoveCmd: sql.NullString{
				String: r.FormValue("RemoveCmd"),
				Valid:  len(r.FormValue("RemoveCmd")) > 0,
			},
		}).WithName("packages-content")
	}))
	// TODO edit package
	// mux.HandleFunc("POST /package/{ID...}", s.HandlePostPackage)
	mux.HandleFunc("DELETE /package/{ID}", s.newHandler(func(r *http.Request) Viewer {
		return s.view.DeletePackage(r.Context(), r.PathValue("ID")).WithName("packages-content")
	}))
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

func (s *Server) HandlePostSettings(w http.ResponseWriter, r *http.Request) {
	plate, err := loadTemplates(s.config.StaticDir)
	if err != nil {
		slog.Error("failed to load templates", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	params := db.UpdateSettingParams{
		Name: r.URL.Query().Get("name"),
	}
	params.Value = r.FormValue(params.Name)
	result := "saved"
	if err = r.ParseForm(); err != nil {
		result = "error"
	}
	if err = s.view.PostSettings(r.Context(), params); err != nil {
		slog.Warn("failed to save settings update", "err", err, "setting", params)
		result = err.Error()
	}
	slog.Info("posted settings", "setting", params)
	if err = plate.ExecuteTemplate(w, "settings-toast", result); err != nil {
		slog.Error("failed to execute settings-toast template", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

type Viewer interface {
	Name() string
}

func (s *Server) newHandler(viewFunc func(r *http.Request) Viewer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		plate, err := loadTemplates(s.config.StaticDir)
		if err != nil {
			slog.Error("failed to load templates", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		view := viewFunc(r)
		if view == nil {
			slog.Error("view finder did not return a view", "method", r.Method, "path", r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err = plate.ExecuteTemplate(w, view.Name(), view); err != nil {
			slog.Error("failed to execute template", "err", err, "name", view.Name())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func loadTemplates(baseDir string) (plate *template.Template, err error) {
	const timeFormat = "2006-01-02T15:04:05Z"
	plate = template.New("").Funcs(template.FuncMap{
		"truncate": func(str string, maxLen int) string {
			maxLen = min(len(str), maxLen)
			return str[0:maxLen]
		},
		"cutOn": func(str, delim string) string {
			return strings.Split(str, delim)[0]
		},
		"lastSeen": func(timeStr string) string {
			t, err := time.Parse(timeFormat, timeStr)
			if err != nil {
				slog.Error("failed to parse timestamp", "err", err, "ts", timeStr)
				return timeStr
			}
			sinceThen := time.Now().UTC().Sub(t)
			if sinceThen < 1*time.Minute {
				return "Connected"
			}
			if sinceThen > 24*time.Hour {
				return fmt.Sprintf("%d days ago", sinceThen/(24*time.Hour))
			}
			return fmt.Sprintf("%s ago", sinceThen)
		},
		"trimPrefix": strings.TrimPrefix,
	})
	plate, err = plate.ParseGlob(path.Join(baseDir, "views/*.html"))
	if err != nil {
		return
	}
	plate, err = plate.ParseGlob(path.Join(baseDir, "templates/*.html"))
	return
}
