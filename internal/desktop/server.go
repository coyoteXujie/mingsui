package desktop

import (
	"context"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/coyoteXujie/mingsui/internal/client"
	"github.com/coyoteXujie/mingsui/internal/config"
)

//go:embed web/*
var webAssets embed.FS

func NewHTTPHandler(app *App) (http.Handler, error) {
	mux := http.NewServeMux()
	assets, err := fs.Sub(webAssets, "web")
	if err != nil {
		return nil, err
	}
	mux.Handle("/", http.FileServer(http.FS(assets)))
	mux.HandleFunc("/api/state", method(http.MethodGet, handleState(app)))
	mux.HandleFunc("/api/config", method(http.MethodPost, handleSaveConfig(app)))
	mux.HandleFunc("/api/start", method(http.MethodPost, handleStart(app)))
	mux.HandleFunc("/api/stop", method(http.MethodPost, handleStop(app)))
	mux.HandleFunc("/api/check", method(http.MethodPost, handleCheck(app)))
	mux.HandleFunc("/api/profile/select", method(http.MethodPost, handleSelectProfile(app)))
	mux.HandleFunc("/api/profile/check", method(http.MethodPost, handleCheckProfile(app)))
	mux.HandleFunc("/api/profiles/import", method(http.MethodPost, handleImportProfiles(app)))
	mux.HandleFunc("/api/subscription", method(http.MethodPost, handleSaveSubscription(app)))
	mux.HandleFunc("/api/subscription/delete", method(http.MethodPost, handleDeleteSubscription(app)))
	mux.HandleFunc("/api/subscription/sync", method(http.MethodPost, handleSyncSubscription(app)))
	return mux, nil
}

type stateResponse struct {
	ConfigPath string               `json:"config_path"`
	Config     config.ClientConfig  `json:"config"`
	Status     client.RuntimeStatus `json:"status"`
}

type messageResponse struct {
	OK      bool                `json:"ok"`
	Message string              `json:"message,omitempty"`
	Health  *client.RelayHealth `json:"health,omitempty"`
	Count   int                 `json:"count,omitempty"`
}

type profileNameRequest struct {
	Name string `json:"name"`
}

type importProfilesRequest struct {
	Content string `json:"content"`
	Replace bool   `json:"replace"`
	Select  string `json:"select"`
}

type subscriptionRequest struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Replace bool   `json:"replace"`
	Select  string `json:"select"`
}

func handleState(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, stateResponse{
			ConfigPath: app.ConfigPath(),
			Config:     app.Config().Redacted(),
			Status:     app.Status(),
		})
	}
}

func handleSaveConfig(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var cfg config.ClientConfig
		if err := readJSON(r, &cfg); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		preserveRedactedSecrets(app.Config(), &cfg)
		if err := app.SaveConfig(cfg); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, messageResponse{OK: true, Message: "配置已保存"})
	}
}

func handleStart(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := app.Start(context.Background()); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, messageResponse{OK: true, Message: "客户端已启动"})
	}
}

func handleStop(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := app.Stop(ctx); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, messageResponse{OK: true, Message: "客户端已停止"})
	}
}

func handleCheck(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		health, err := app.CheckRelayStatus(ctx)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, messageResponse{OK: true, Message: "relay 可连接", Health: &health})
	}
}

func handleSelectProfile(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req profileNameRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := app.SelectRelayProfile(req.Name); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, messageResponse{OK: true, Message: "profile 已选择"})
	}
}

func handleCheckProfile(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req profileNameRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		health, err := app.CheckRelayProfileStatus(ctx, req.Name)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, messageResponse{OK: true, Message: "profile 可连接", Health: &health})
	}
}

func handleImportProfiles(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req importProfilesRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if strings.TrimSpace(req.Content) == "" {
			writeErrorMessage(w, http.StatusBadRequest, "订阅内容不能为空")
			return
		}
		count, err := app.ImportRelayProfiles([]byte(req.Content), req.Replace, req.Select)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, messageResponse{OK: true, Message: "profile 已导入", Count: count})
	}
}

func handleSaveSubscription(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req subscriptionRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := app.UpsertRelaySubscription(config.RelaySubscription{Name: req.Name, URL: req.URL}, req.Replace); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, messageResponse{OK: true, Message: "订阅已保存"})
	}
}

func handleDeleteSubscription(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req profileNameRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := app.RemoveRelaySubscription(req.Name); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, messageResponse{OK: true, Message: "订阅已删除"})
	}
}

func handleSyncSubscription(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req subscriptionRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		count, err := app.SyncRelaySubscription(ctx, req.Name, req.Replace, req.Select)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, messageResponse{OK: true, Message: "订阅已同步", Count: count})
	}
}

func method(want string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != want {
			w.Header().Set("Allow", want)
			writeErrorMessage(w, http.StatusMethodNotAllowed, "方法不允许")
			return
		}
		next(w, r)
	}
}

func readJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeErrorMessage(w, status, err.Error())
}

func writeErrorMessage(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, messageResponse{OK: false, Message: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func preserveRedactedSecrets(current config.ClientConfig, next *config.ClientConfig) {
	if next.Token == config.RedactedValue {
		next.Token = current.Token
	}
	if next.LocalAuth.Password == config.RedactedValue {
		next.LocalAuth.Password = current.LocalAuth.Password
	}
	for i := range next.Profiles {
		if next.Profiles[i].Token != config.RedactedValue {
			continue
		}
		for _, profile := range current.Profiles {
			if profile.Name == next.Profiles[i].Name {
				next.Profiles[i].Token = profile.Token
				break
			}
		}
	}
	for i := range next.Subscriptions {
		if next.Subscriptions[i].URL != config.RedactedValue {
			continue
		}
		for _, sub := range current.Subscriptions {
			if sub.Name == next.Subscriptions[i].Name {
				next.Subscriptions[i].URL = sub.URL
				break
			}
		}
	}
}
