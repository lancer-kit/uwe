package uwe

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
)

// todo: rename
type AdminConfig struct {
	PPROF        bool   `json:"pprof" yaml:"pprof"`
	AllowControl bool   `json:"allow_control" yaml:"allow_control"`
	Host         string `json:"host" yaml:"host"`
	Port         int    `json:"port" yaml:"port"`
}

func (chief *Chief) getRouter() http.Handler {

	mux := http.NewServeMux()
	//if chief.AdminConfig.PPROF {
	//	mux.Handle("/pprof", pprof.Handler(chief.AppName+"_pprof"))
	//}

	mux.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = w.Write([]byte("Not Allowed"))
			return
		}

		data := chief.wPool.GetWorkersStates()
		tmpl, _ := template.New("data").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>UWE</title>
</head>
<body>
<table>
  <tr>
    <th>Worker</th>
    <th>Status</th>
    <th>Action</th>
  </tr>
    {{ range $key, $value := . }}
      <tr>
        <td>{{ $key }}</td>
        <td>{{ $value }}</td>
        <td> NA</td>
      </tr>
    {{ end }}
</table>
</body>
</html>
`)
		_ = tmpl.Execute(w, data)
	})

	return mux
}

func (chief *Chief) adminAPI(wCtx context.Context) ExitCode {
	addr := fmt.Sprintf("%s:%d", chief.AdminConfig.Host, chief.AdminConfig.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: chief.getRouter(),
	}

	serverFailed := make(chan struct{})
	go func() {
		chief.logger.Info("Starting API Server at: ", addr)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			chief.logger.WithError(err).Error("server failed")
			close(serverFailed)
		}
	}()

	select {
	case <-wCtx.Done():
		chief.logger.Info("Shutting down the API Server...")
		serverCtx, _ := context.WithTimeout(context.Background(), ForceStopTimeout)

		err := server.Shutdown(serverCtx)
		if err != nil {
			chief.logger.Info("Api Server gracefully stopped")
		}

		chief.logger.Info("Api Server gracefully stopped")
		return ExitCodeOk
	case <-serverFailed:
		return ExitCodeFailed
	}
}
