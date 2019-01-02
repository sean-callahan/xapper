package xapper

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func Router() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/", index)
	r.HandleFunc("/{device}", State)
	r.HandleFunc("/{device}/{group}/{channel}/gain", GainAdjust)
	r.HandleFunc("/{device}/{group}/{channel}/mute", Mute)

	return r
}

func index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

}

func device(vars map[string]string) (*Device, error) {
	id, err := strconv.Atoi(vars["device"])
	if err != nil {
		return nil, err
	}
	if id < 0 || id > 8 {
		return nil, errors.New("device id out of range")
	}

	return Global.Devices[id], nil
}

func State(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	vars := mux.Vars(r)
	d, err := device(vars)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, channels := range d.Channels {
		for _, ch := range channels {
			ch.mu.RLock()
			defer ch.mu.RUnlock()
		}
	}

	json.NewEncoder(w).Encode(d)
}

func GainAdjust(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	d, err := device(vars)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value, err := strconv.ParseInt(r.FormValue("value"), 10, 32)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	channelId, err := strconv.Atoi(vars["channel"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ch := d.Channels[Group(vars["group"])][channelId-1]

	gain, err := ch.SetGain(float32(value))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, gain)
}

func Mute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	d, err := device(vars)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mute, err := strconv.ParseBool(r.FormValue("value"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	channelId, err := strconv.Atoi(vars["channel"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ch := d.Channels[Group(vars["group"])][channelId-1]

	muted, err := ch.Mute(mute)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(strconv.FormatBool(muted)))
}
