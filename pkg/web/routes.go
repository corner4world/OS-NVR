// Copyright 2020-2021 The OS-NVR Authors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; version 2.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"nvr/pkg/group"
	"nvr/pkg/log"
	"nvr/pkg/monitor"
	"nvr/pkg/storage"
	"nvr/pkg/system"
	"nvr/pkg/web/auth"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

const redirect = `
	<head><script>
		window.location.href = window.location.href.replace("logout", "live")
	</script></head>`

// Logout prompts for login and redirects. Old login should be overwritten.
func Logout() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("Authorization") {
		case "Basic Og==":
		case "":
		default:
			w.Header().Set("WWW-Authenticate", `Basic realm=""`)
			http.Error(w, "", http.StatusUnauthorized)
			return
		}

		if _, err := io.WriteString(w, redirect); err != nil {
			http.Error(w, "could not write string", http.StatusInternalServerError)
			return
		}
	})
}

// Static serves files from `web/static`.
func Static(path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}
		// w.Header().Set("Cache-Control", "max-age=2629800")
		w.Header().Set("Cache-Control", "no-cache")

		h := http.StripPrefix("/static/", http.FileServer(http.Dir(path)))
		h.ServeHTTP(w, r)
	})
}

// Storage serves files from `web/static`.
func Storage(path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}
		h := http.StripPrefix("/storage/", http.FileServer(http.Dir(path)))
		h.ServeHTTP(w, r)
	})
}

// HLS serves files from shmHLS.
func HLS(env *storage.ConfigEnv) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")

		h := http.StripPrefix("/hls/", http.FileServer(http.Dir(env.SHMhls())))
		h.ServeHTTP(w, r)
	})
}

// Status returns system status.
func Status(sys *system.System) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(sys.Status()); err != nil {
			http.Error(w, "could not encode json", http.StatusInternalServerError)
		}
	})
}

// TimeZone returns system timeZone.
func TimeZone(timeZone string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(timeZone); err != nil {
			http.Error(w, "could not encode json", http.StatusInternalServerError)
		}
	})
}

// General handler returns general configuration in json format.
func General(general *storage.ConfigGeneral) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}

		j, err := json.Marshal(general.Get())
		if err != nil {
			http.Error(w, "failed to marshal general config", http.StatusInternalServerError)
			return
		}
		if _, err := w.Write(j); err != nil {
			http.Error(w, "could not write data", http.StatusInternalServerError)
		}
	})
}

// GeneralSet handler to set general configuration.
func GeneralSet(general *storage.ConfigGeneral) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		var config storage.GeneralConfig
		if err = json.Unmarshal(body, &config); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if config.DiskSpace == "" {
			http.Error(w, "DiskSpace missing", http.StatusBadRequest)
			return
		}

		err = general.Set(config)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

// Users returns a censored user list in json format.
func Users(a *auth.Authenticator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}
		j, err := json.Marshal(a.UsersList())
		if err != nil {
			http.Error(w, "failed to marshal user list", http.StatusInternalServerError)
			return
		}
		if _, err := w.Write(j); err != nil {
			http.Error(w, "could not write data", http.StatusInternalServerError)
		}
	})
}

// UserSet handler to set user details.
func UserSet(a *auth.Authenticator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		var user auth.Account
		if err = json.Unmarshal(body, &user); err != nil {
			http.Error(w, "unmarshal error: "+err.Error(), http.StatusBadRequest)
			return
		}

		err = a.UserSet(user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	})
}

// UserDelete handler to delete user.
func UserDelete(a *auth.Authenticator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}

		name := r.URL.Query().Get("id")
		if name == "" {
			http.Error(w, "id missing", http.StatusBadRequest)
			return
		}

		err := a.UserDelete(name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

// MonitorList returns a censored monitor list with ID, Name and CaptureAudio.
func MonitorList(monitorList func() monitor.Configs) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}
		u, err := json.Marshal(monitorList())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := w.Write(u); err != nil {
			http.Error(w, "could not write data", http.StatusInternalServerError)
			return
		}
	})
}

// MonitorConfigs returns monitor configurations in json format.
func MonitorConfigs(c *monitor.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}
		u, err := json.Marshal(c.MonitorConfigs())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := w.Write(u); err != nil {
			http.Error(w, "could not write data", http.StatusInternalServerError)
			return
		}
	})
}

// MonitorRestart handler to restart monitor.
func MonitorRestart(m *monitor.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}

		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id missing", http.StatusBadRequest)
			return
		}

		monitor, exists := m.Monitors[id]
		if !exists {
			http.Error(w, "monitor does not exist", http.StatusBadRequest)
			return
		}

		monitor.Stop()
		if err := monitor.Start(); err != nil {
			http.Error(w, "could not restart monitor: "+err.Error(), http.StatusInternalServerError)
		}
	})
}

// MonitorSet handler to set monitor configuration.
func MonitorSet(c *monitor.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		var m monitor.Config
		if err = json.Unmarshal(body, &m); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := checkIDandName(m); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = c.MonitorSet(m["id"], m)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

// MonitorDelete handler to delete monitor.
func MonitorDelete(m *monitor.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}

		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id missing", http.StatusBadRequest)
			return
		}

		err := m.MonitorDelete(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

// GroupConfigs returns group configurations in json format.
func GroupConfigs(m *group.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}
		u, err := json.Marshal(m.GroupConfigs())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := w.Write(u); err != nil {
			http.Error(w, "could not write data", http.StatusInternalServerError)
			return
		}
	})
}

// GroupSet handler to set group configuration.
func GroupSet(m *group.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		var g group.Config
		if err = json.Unmarshal(body, &g); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := checkIDandName(g); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err = m.GroupSet(g["id"], g); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

// ErrEmptyValue value cannot be empty.
var ErrEmptyValue = errors.New("value cannot be empty")

// ErrContainsSpaces value cannot contain spaces.
var ErrContainsSpaces = errors.New("value cannot contain spaces")

func checkIDandName(input map[string]string) error {
	switch {
	case input["id"] == "":
		return fmt.Errorf("id: %w", ErrEmptyValue)
	case containsSpaces(input["id"]):
		return fmt.Errorf("id: %w", ErrContainsSpaces)
	case input["name"] == "":
		return fmt.Errorf("name: %w", ErrEmptyValue)
	case containsSpaces(input["name"]):
		return fmt.Errorf("name. %w", ErrContainsSpaces)
	default:
		return nil
	}
}

// GroupDelete handler to delete group.
func GroupDelete(m *group.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}

		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id missing", http.StatusBadRequest)
			return
		}

		err := m.GroupDelete(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

// RecordingQuery handles recording queries.
// TODO: Replace api with: time, limit, reverse, monitors[].
func RecordingQuery(crawler *storage.Crawler, log *log.Logger) http.Handler { //nolint:funlen
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}
		query := r.URL.Query()
		limit := query.Get("limit")
		if limit == "" {
			http.Error(w, "limit missing", http.StatusBadRequest)
			return
		}

		limitInt, err := strconv.Atoi(limit)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not convert limit to int: %v", err), http.StatusBadRequest)
			return
		}

		before := query.Get("before")
		if before == "" {
			http.Error(w, "before missing", http.StatusBadRequest)
			return
		}
		if len(before) < 19 {
			http.Error(w, "before to short", http.StatusBadRequest)
			return
		}
		reverse := query.Get("reverse")

		monitorsCSV := query.Get("monitors")

		var monitors []string
		if monitorsCSV != "" {
			monitors = strings.Split(monitorsCSV, ",")
		}

		q := &storage.CrawlerQuery{
			Time:     before,
			Limit:    limitInt,
			Reverse:  reverse == "true",
			Monitors: monitors,
		}

		recordings, err := crawler.RecordingByQuery(q)
		if err != nil {
			log.Error().Src("storage").
				Msgf("crawler: could not process recording query: %v", err)

			http.Error(w, "could not process recording query", http.StatusInternalServerError)
			return
		}

		u, err := json.Marshal(recordings)
		if err != nil {
			http.Error(w, "could not marshal data", http.StatusInternalServerError)
			return
		}

		if _, err := w.Write(u); err != nil {
			http.Error(w, "could not write data", http.StatusInternalServerError)
			return
		}
	})
}

// LogFeed opens a websocket with system logs.
func LogFeed(l *log.Logger, a *auth.Authenticator) http.Handler { //nolint:funlen,gocognit
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}
		query := r.URL.Query()

		levelsCSV := query.Get("levels")
		var levels []log.Level
		if levelsCSV != "" {
			for _, levelStr := range strings.Split(levelsCSV, ",") {
				levelInt, err := strconv.Atoi(levelStr)
				if err != nil {
					http.Error(w,
						fmt.Sprintf("invalid levels list: %v %v", levelsCSV, err),
						http.StatusBadRequest)
				}
				levels = append(levels, log.Level(levelInt))
			}
		}

		sourcesCSV := query.Get("sources")
		var sources []string
		if sourcesCSV != "" {
			sources = strings.Split(sourcesCSV, ",")
		}

		q := log.Query{
			Levels:  levels,
			Sources: sources,
		}

		upgrader := websocket.Upgrader{}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer c.Close()

		feed, cancel := l.Subscribe()
		defer cancel()

		authHeader := r.Header.Get("Authorization")
		for {
			log := <-feed

			levelMatching := false
			for _, level := range q.Levels {
				if level == log.Level {
					levelMatching = true
					break
				}
			}
			sourceMatching := false
			for _, src := range q.Sources {
				if src == log.Src {
					sourceMatching = true
					break
				}
			}
			if !levelMatching || !sourceMatching {
				continue
			}

			// Validate auth before each message.
			auth := a.ValidateAuth(authHeader)
			if !auth.IsValid || !auth.User.IsAdmin {
				return
			}

			raw, err := json.Marshal(log)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			if err := c.WriteMessage(websocket.TextMessage, raw); err != nil {
				return
			}
		}
	})
}

// LogQuery handles log queries.
func LogQuery(l *log.Logger) http.Handler { //nolint:funlen
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}
		query := r.URL.Query()

		limit := query.Get("limit")
		if limit == "" {
			http.Error(w, "limit missing", http.StatusBadRequest)
			return
		}

		limitInt, err := strconv.Atoi(limit)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not convert limit to int: %v", err), http.StatusBadRequest)
			return
		}

		levelsCSV := query.Get("levels")
		var levels []log.Level
		if levelsCSV != "" {
			for _, levelStr := range strings.Split(levelsCSV, ",") {
				levelInt, err := strconv.Atoi(levelStr)
				if err != nil {
					http.Error(w,
						fmt.Sprintf("invalid levels list: %v %v", levelsCSV, err),
						http.StatusBadRequest)
				}
				levels = append(levels, log.Level(levelInt))
			}
		}

		sourcesCSV := query.Get("sources")
		var sources []string
		if sourcesCSV != "" {
			sources = strings.Split(sourcesCSV, ",")
		}

		time := query.Get("time")
		timeInt, err := strconv.Atoi(time)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not convert limit to int: %v", err), http.StatusBadRequest)
			return
		}

		q := log.Query{
			Levels:  levels,
			Sources: sources,
			Time:    log.UnixMillisecond(timeInt),
			Limit:   limitInt,
		}

		logs, err := l.Query(q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		logsJSON, err := json.Marshal(logs)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not marshal data: %v", err), http.StatusInternalServerError)
			return
		}

		if _, err := w.Write(logsJSON); err != nil {
			http.Error(w, fmt.Sprintf("could not write data: %v", err), http.StatusInternalServerError)
			return
		}
	})
}

// LogSources handles list of log sources.
func LogSources(l *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
			return
		}

		sources, err := json.Marshal(l.Sources())
		if err != nil {
			http.Error(w, fmt.Sprintf("could not marshal data: %v", err), http.StatusInternalServerError)
			return
		}

		if _, err := w.Write(sources); err != nil {
			http.Error(w, fmt.Sprintf("could not write data: %v", err), http.StatusInternalServerError)
			return
		}
	})
}

func containsSpaces(s string) bool {
	return strings.Contains(s, " ")
}
