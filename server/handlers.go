package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"slices"
	"sort"

	"src.acicovic.me/divelog/server/utils"
)

const (
	PathFavicon      = "/favicon.ico"
	PathProzaLibre   = "/ProzaLibre-Regular.woff2"
	FileFavicon      = "data/favicon.ico"
	FileProzaLibre   = "data/ProzaLibre-Regular.woff2"
	ContentTypeWoff2 = "font/woff2"
)

var _pageTemplate = template.Must(template.ParseFiles("data/pagetemplate.html"))

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	var filePath, contentType string
	switch r.URL.Path {
	case PathFavicon:
		filePath = FileFavicon
	case PathProzaLibre:
		filePath = FileProzaLibre
		contentType = ContentTypeWoff2
	default:
		http.NotFound(w, r)
		return
	}

	var (
		file *os.File
		fi   os.FileInfo
	)
	file, err := os.Open(filePath)
	if err == nil {
		fi, err = file.Stat()
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	http.ServeContent(w, r, r.URL.Path[1:], fi.ModTime(), file)
}

func fetchSites(w http.ResponseWriter, r *http.Request) {
	var (
		resp []byte
		err  error
	)

	if r.URL.Query().Get("headonly") == "true" {
		heads := make([]*SiteHead, 0, len(bluefin.DiveSites))
		for _, site := range bluefin.DiveSites[1:] {
			heads = append(heads, &SiteHead{
				ID:   site.ID,
				Name: site.Name,
			})
		}
		sort.Slice(heads, func(i, j int) bool {
			return heads[i].Name < heads[j].Name
		})
		resp, err = json.Marshal(heads)
	} else {
		sites := []*SiteFull{}
		for _, site := range bluefin.DiveSites[1:] {
			sites = append(sites, NewSiteFull(site, bluefin.Dives[1:]))
		}
		resp, err = json.Marshal(sites)
	}

	if err != nil {
		trace(_error, "http: failed to marshal dive site data: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	send(w, resp)
}

func fetchSite(w http.ResponseWriter, r *http.Request) {
	siteID := utils.ConvertAndCheckID(r.PathValue("id"), bluefin.LargestSiteID())
	if siteID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	site := bluefin.DiveSites[siteID]

	resp, err := json.Marshal(NewSiteFull(site, bluefin.Dives[1:]))
	if err != nil {
		trace(_error, "http: failed to marshal single dive site data: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	send(w, resp)
}

func fetchTrips(w http.ResponseWriter, r *http.Request) {
	trips := make([]*Trip, 0, len(bluefin.DiveTrips))
	reverse := r.URL.Query().Get("reverse") == "true"
	if reverse {
		for _, trip := range bluefin.DiveTrips[1:] {
			trips = append(trips, &Trip{
				ID:    trip.ID,
				Label: trip.Label,
			})
		}
	} else {
		for i := len(bluefin.DiveTrips) - 1; i > 0; i-- {
			trips = append(trips, &Trip{
				ID:    bluefin.DiveTrips[i].ID,
				Label: bluefin.DiveTrips[i].Label,
			})
		}
	}

	for _, trip := range trips {
		for _, dive := range bluefin.Dives[1:] {
			if dive.DiveTripID == trip.ID {
				trip.LinkedDives = append(trip.LinkedDives, NewDiveHead(dive, bluefin.DiveSites[dive.DiveSiteID]))
			}
		}
		if !reverse {
			slices.Reverse(trip.LinkedDives)
		}
	}

	resp, err := json.Marshal(trips)
	if err != nil {
		trace(_error, "http: failed to marshal dive trip data: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	send(w, resp)
}

func fetchDives(w http.ResponseWriter, r *http.Request) {
	var (
		resp []byte
		err  error
		tag  = r.URL.Query().Get("tag")
	)

	if r.URL.Query().Get("headonly") == "true" {
		heads := make([]*DiveHead, 0, len(bluefin.Dives))
		for _, dive := range bluefin.Dives[1:] {
			heads = append(heads, NewDiveHead(dive, bluefin.DiveSites[dive.DiveSiteID]))
		}
		resp, err = json.Marshal(heads)
	} else {
		dives := []*DiveFull{}
		for _, dive := range bluefin.Dives[1:] {
			if dive.IsTaggedWith(tag) {
				dives = append(dives, NewDiveFull(dive, bluefin.DiveSites[dive.DiveSiteID]))
			}
		}
		resp, err = json.Marshal(dives)
	}

	if err != nil {
		trace(_error, "http: failed to marshal dive data: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	send(w, resp)
}

func fetchDive(w http.ResponseWriter, r *http.Request) {
	diveID := utils.ConvertAndCheckID(r.PathValue("id"), bluefin.LargestDiveID())
	if diveID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	dive := bluefin.Dives[diveID]

	resp, err := json.Marshal(NewDiveFull(dive, bluefin.DiveSites[dive.DiveSiteID]))
	if err != nil {
		trace(_error, "http: failed to marshal single dive data: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	send(w, resp)
}

func fetchTags(w http.ResponseWriter, r *http.Request) {
	tags := make(map[string]int)
	for _, dive := range bluefin.Dives[1:] {
		for _, tag := range dive.Tags {
			tags[tag]++
		}
	}

	resp, err := json.Marshal(tags)
	if err != nil {
		trace(_error, "http: failed to marshal tags data: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	send(w, resp)
}

func renderDives(w http.ResponseWriter, r *http.Request) {
	trips := make([]*Trip, 0, len(bluefin.DiveTrips))
	for i := len(bluefin.DiveTrips) - 1; i > 0; i-- {
		trip := &Trip{
			ID:    i,
			Label: bluefin.DiveTrips[i].Label,
		}
		for i := len(bluefin.Dives) - 1; i > 0; i-- {
			dive := bluefin.Dives[i]
			if dive.DiveTripID == trip.ID {
				trip.LinkedDives = append(
					trip.LinkedDives,
					NewDiveHead(dive, bluefin.DiveSites[dive.DiveSiteID]),
				)
			}
		}
		trips = append(trips, trip)
	}

	renderTemplate(w, Page{
		Title:      "Dives",
		Supertitle: "All",
		Trips:      trips,
	})
}

func renderSites(w http.ResponseWriter, r *http.Request) {
	heads := make([]*SiteHead, 0, len(bluefin.DiveSites))
	for _, site := range bluefin.DiveSites[1:] {
		heads = append(heads, &SiteHead{
			ID:   site.ID,
			Name: site.Name,
		})
	}
	sort.Slice(heads, func(i, j int) bool {
		return heads[i].Name < heads[j].Name
	})

	renderTemplate(w, Page{
		Title:      "Dive sites",
		Supertitle: "All",
		Sites:      heads,
	})
}

func renderDive(w http.ResponseWriter, r *http.Request) {
	diveID := utils.ConvertAndCheckID(r.PathValue("id"), bluefin.LargestDiveID())
	if diveID == 0 {
		renderNotFound(w, "dive not found")
		return
	}
	dive := bluefin.Dives[diveID]
	site := bluefin.DiveSites[dive.DiveSiteID]

	page := Page{
		Title:      site.Name,
		Supertitle: fmt.Sprintf("Dive %d", dive.Number),
		Dive:       NewDiveFull(dive, site),
	}
	// fix it here because this is the only scenario where it's needed
	// (although it's not a good design)
	if page.Dive.NextID == len(bluefin.Dives) {
		page.Dive.NextID = 0
	}

	renderTemplate(w, page)
}

func renderSite(w http.ResponseWriter, r *http.Request) {
	siteID := utils.ConvertAndCheckID(r.PathValue("id"), bluefin.LargestSiteID())
	if siteID == 0 {
		renderNotFound(w, "site not found")
		return
	}
	site := bluefin.DiveSites[siteID]

	renderTemplate(w, Page{
		Title:      site.Name,
		Supertitle: site.Coordinates,
		Site:       NewSiteFull(site, bluefin.Dives[1:]),
	})
}

func renderTags(w http.ResponseWriter, r *http.Request) {
	tags := make(map[string]int)
	for _, dive := range bluefin.Dives[1:] {
		for _, tag := range dive.Tags {
			tags[tag]++
		}
	}

	renderTemplate(w, Page{
		Title:      "Tags",
		Supertitle: "All",
		Tags:       tags,
	})
}

func renderTaggedDives(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	dives := []*DiveHead{}
	for i := len(bluefin.Dives) - 1; i > 0; i-- {
		dive := bluefin.Dives[i]
		for _, t := range dive.Tags {
			if t == tag {
				dives = append(
					dives,
					NewDiveHead(dive, bluefin.DiveSites[dive.DiveSiteID]),
				)
			}
		}
	}

	if len(dives) == 0 {
		renderNotFound(w, "")
		return
	}

	renderTemplate(w, Page{
		Title:      tag,
		Supertitle: "Dives tagged with",
		Dives:      dives,
	})
}

func renderNotFound(w http.ResponseWriter, title string) {
	if title == "" {
		title = "not found"
	}

	renderTemplate(w, Page{
		Title:      title,
		Supertitle: "404",
		NotFound:   true,
	})
}

// TODO: align this with how it is done in koi
func multiplexer() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /hms/dives", renderDives)
	trace(_https, "handler registered for /hms/dives")

	mux.HandleFunc("GET /hms/dives/{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/hms/dives", http.StatusMovedPermanently)
	})
	trace(_https, "handler registered for /hms/dives/")

	mux.HandleFunc("GET /hms/sites", renderSites)
	trace(_https, "handler registered for /hms/sites")

	mux.HandleFunc("GET /hms/sites/{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/hms/sites", http.StatusMovedPermanently)
	})
	trace(_https, "handler registered for /hms/sites/")

	mux.HandleFunc("GET /hms/tags", renderTags)
	trace(_https, "handler registered for /hms/tags")

	mux.HandleFunc("GET /hms/tags/{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/hms/tags", http.StatusMovedPermanently)
	})
	trace(_https, "handler registered for /hms/tags/")

	mux.HandleFunc("GET /hms/dives/{id}", renderDive)
	trace(_https, "handler registered for /hms/dives/{id}")

	mux.HandleFunc("GET /hms/sites/{id}", renderSite)
	trace(_https, "handler registered for /hms/sites/{id}")

	mux.HandleFunc("GET /hms/tags/{tag}", renderTaggedDives)
	trace(_https, "handler registered for /hms/tags/{tag}")

	mux.HandleFunc("GET /hms/about", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, Page{
			Title:      "this site",
			Supertitle: "about",
			About:      true,
		})
	})
	trace(_https, "handler registered for /hms/about")

	// data handlers
	mux.HandleFunc("GET /data/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	trace(_https, "handler registered for /data/")

	mux.HandleFunc("GET /data/sites", fetchSites)
	trace(_https, "handler registered for /data/sites")
	// DEVNOTE: /data/sites/{$} returns 404

	mux.HandleFunc("GET /data/sites/{id}", fetchSite)
	trace(_https, "handler registered for /data/sites/{id}")

	mux.HandleFunc("GET /data/trips", fetchTrips)
	trace(_https, "handler registered for /data/trips")
	// DEVNOTE: /data/trips/{$} returns 404

	mux.HandleFunc("GET /data/dives", fetchDives)
	trace(_https, "handler registered for /data/dives")
	// DEVNOTE: /data/dives/{$} returns 404

	mux.HandleFunc("GET /data/dives/{id}", fetchDive)
	trace(_https, "handler registered for /data/dives/{id}")

	mux.HandleFunc("GET /data/tags", fetchTags)
	trace(_https, "handler registered for /data/tags")
	// DEVNOTE: /data/tags/{$} returns 404

	mux.HandleFunc("GET /", defaultHandler)
	trace(_https, "handler registered for /")

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/hms/dives", http.StatusMovedPermanently)
	})
	trace(_https, "handler registered for /{$}")

	// local API handlers
	if _serverControl.localAPI {
		mux.HandleFunc("GET /data/0", fetchAll)
		trace(_https, "handler registered for /data/0")

		mux.HandleFunc("POST /action/fail", forceFailure)
		trace(_https, "handler registered for /action/fail")

		mux.HandleFunc("POST /action/rebuild", rebuildDatabase)
		trace(_https, "handler registered for /action/rebuild")
	}

	return mux
}

func send(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write(data)
	if err != nil {
		trace(_error, "http: send: %v", err)
	}
}

func renderTemplate(w http.ResponseWriter, p Page) {
	if !p.check() {
		trace(_error, "http: incorrect internal page state")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := _pageTemplate.Execute(w, p); err != nil {
		trace(_error, "http: render template: %v", err)
	}
}
