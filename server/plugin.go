package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	// BotId of the created bot account.
	botID string
}

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	switch path := r.URL.Path; path {
	case "/api/v1/settings":
		p.httpMeetingSettings(w, r)
	default:
		http.NotFound(w, r)
	}
}

// OnActivate
func (p *Plugin) OnActivate() error {
	if err := p.registerCommands(); err != nil {
		return errors.Wrap(err, "failed to register commands")
	}

	botID, err := p.Helpers.EnsureBot(&model.Bot{
		Username:    "agenda",
		DisplayName: "Agenda Plugin Bot",
		Description: "Created by the Agenda plugin.",
	})
	if err != nil {
		return errors.Wrap(err, "failed to ensure agenda bot")
	}
	p.botID = botID

	return nil
}

func (p *Plugin) httpMeetingSettings(w http.ResponseWriter, r *http.Request) {

	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	if mattermostUserId == "" {
		http.Error(w, "Not Authorized", http.StatusUnauthorized)
	}

	switch r.Method {
	case http.MethodPost:
		p.httpMeetingSaveSettings(w, r, mattermostUserId)
	case http.MethodGet:
		p.httpMeetingGetSettings(w, r, mattermostUserId)
	default:
		http.Error(w, "Request: "+r.Method+" is not allowed.", http.StatusMethodNotAllowed)
	}
}

func (p *Plugin) httpMeetingSaveSettings(w http.ResponseWriter, r *http.Request, mmUserId string) {

	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var meeting *Meeting
	if err = json.Unmarshal(body, &meeting); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = p.SaveMeeting(meeting); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("{\"status\": \"OK\"}"))
}

func (p *Plugin) httpMeetingGetSettings(w http.ResponseWriter, r *http.Request, mmUserId string) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	channelId, ok := r.URL.Query()["channelId"]

	if !ok || len(channelId[0]) < 1 {
		http.Error(w, "Missing channelId parameter", http.StatusBadRequest)
		return
	}

	meeting, err := p.GetMeeting(channelId[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := json.Marshal(meeting)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(resp)
}
