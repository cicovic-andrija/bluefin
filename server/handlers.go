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
	PathStyle        = "/style.css"
	FileFavicon      = "data" + PathFavicon
	FileProzaLibre   = "data" + PathProzaLibre
	FileStyle        = "data" + PathStyle
	ContentTypeWoff2 = "font/woff2"
	ContentTypeCSS   = "text/css"
)

var _page_template = template.Must(template.ParseFiles("data/pagetemplate.html"))

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	var filePath, contentType string
	switch r.URL.Path {
	case PathFavicon:
		filePath = FileFavicon
	case PathProzaLibre:
		filePath = FileProzaLibre
		contentType = ContentTypeWoff2
	case PathStyle:
		filePath = FileStyle
		contentType = ContentTypeCSS
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

func fetchSites(w http.ResponseWriter, r *http.Request, divelog *DiveLog) {
	var (
		resp []byte
		err  error
	)

	if r.URL.Query().Get("headonly") == "true" {
		heads := make([]*SiteHead, 0, len(divelog.DiveSites))
		for _, site := range divelog.DiveSites[1:] {
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
		for _, site := range divelog.DiveSites[1:] {
			sites = append(sites, NewSiteFull(site, divelog.Dives[1:]))
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

func fetchSite(w http.ResponseWriter, r *http.Request, divelog *DiveLog) {
	siteID := utils.ConvertAndCheckID(r.PathValue("id"), divelog.LargestSiteID())
	if siteID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	site := divelog.DiveSites[siteID]

	resp, err := json.Marshal(NewSiteFull(site, divelog.Dives[1:]))
	if err != nil {
		trace(_error, "http: failed to marshal single dive site data: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	send(w, resp)
}

func fetchTrips(w http.ResponseWriter, r *http.Request, divelog *DiveLog) {
	trips := make([]*Trip, 0, len(divelog.DiveTrips))
	reverse := r.URL.Query().Get("reverse") == "true"
	if reverse {
		for _, trip := range divelog.DiveTrips[1:] {
			trips = append(trips, &Trip{
				ID:    trip.ID,
				Label: trip.Label,
			})
		}
	} else {
		for i := len(divelog.DiveTrips) - 1; i > 0; i-- {
			trips = append(trips, &Trip{
				ID:    divelog.DiveTrips[i].ID,
				Label: divelog.DiveTrips[i].Label,
			})
		}
	}

	for _, trip := range trips {
		for _, dive := range divelog.Dives[1:] {
			if dive.DiveTripID == trip.ID {
				trip.LinkedDives = append(trip.LinkedDives, NewDiveHead(dive, divelog.DiveSites[dive.DiveSiteID]))
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

func fetchDives(w http.ResponseWriter, r *http.Request, divelog *DiveLog) {
	var (
		resp []byte
		err  error
		tag  = r.URL.Query().Get("tag")
	)

	if r.URL.Query().Get("headonly") == "true" {
		heads := make([]*DiveHead, 0, len(divelog.Dives))
		for _, dive := range divelog.Dives[1:] {
			heads = append(heads, NewDiveHead(dive, divelog.DiveSites[dive.DiveSiteID]))
		}
		resp, err = json.Marshal(heads)
	} else {
		dives := []*DiveFull{}
		for _, dive := range divelog.Dives[1:] {
			if dive.IsTaggedWith(tag) {
				dives = append(dives, NewDiveFull(dive, divelog.DiveSites[dive.DiveSiteID]))
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

func fetchDive(w http.ResponseWriter, r *http.Request, divelog *DiveLog) {
	diveID := utils.ConvertAndCheckID(r.PathValue("id"), divelog.LargestDiveID())
	if diveID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	dive := divelog.Dives[diveID]

	resp, err := json.Marshal(NewDiveFull(dive, divelog.DiveSites[dive.DiveSiteID]))
	if err != nil {
		trace(_error, "http: failed to marshal single dive data: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	send(w, resp)
}

func fetchTags(w http.ResponseWriter, r *http.Request, divelog *DiveLog) {
	tags := make(map[string]int)
	for _, dive := range divelog.Dives[1:] {
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

// TODO: This function can be refactored to be similar to renderSites.
func renderDives(w http.ResponseWriter, r *http.Request, divelog *DiveLog) {
	trips := make([]*Trip, 0, len(divelog.DiveTrips))
	for i := len(divelog.DiveTrips) - 1; i > 0; i-- {
		trip := &Trip{
			ID:    i,
			Label: divelog.DiveTrips[i].Label,
		}
		for i := len(divelog.Dives) - 1; i > 0; i-- {
			dive := divelog.Dives[i]
			if dive.DiveTripID == trip.ID {
				trip.LinkedDives = append(
					trip.LinkedDives,
					NewDiveHead(dive, divelog.DiveSites[dive.DiveSiteID]),
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

func renderSites(w http.ResponseWriter, r *http.Request, divelog *DiveLog) {
	regionMap := make(map[string][]*SiteHead)
	for _, site := range divelog.DiveSites[1:] {
		regionMap[site.Region] = append(regionMap[site.Region], &SiteHead{
			ID:   site.ID,
			Name: site.Name,
		})
	}

	siteHeads := make([]*GroupedSites, 0, len(regionMap))
	for region, sites := range regionMap {
		sort.Slice(sites, func(i, j int) bool {
			return sites[i].Name < sites[j].Name
		})
		siteHeads = append(siteHeads, &GroupedSites{
			Region:      region,
			LinkedSites: sites,
		})
	}

	sort.Slice(siteHeads, func(i, j int) bool {
		return siteHeads[i].Region < siteHeads[j].Region
	})

	renderTemplate(w, Page{
		Title:        "Dive sites",
		Supertitle:   "All",
		GroupedSites: siteHeads,
	})
}

func renderDive(w http.ResponseWriter, r *http.Request, divelog *DiveLog) {
	diveID := utils.ConvertAndCheckID(r.PathValue("id"), divelog.LargestDiveID())
	if diveID == 0 {
		renderNotFound(w, "dive not found")
		return
	}
	dive := divelog.Dives[diveID]
	site := divelog.DiveSites[dive.DiveSiteID]

	page := Page{
		Title:      site.Name,
		Supertitle: fmt.Sprintf("Dive %d", dive.Number),
		Dive:       NewDiveFull(dive, site),
	}
	// fix it here because this is the only scenario where it's needed
	// (although it's not a good design)
	if page.Dive.NextID == len(divelog.Dives) {
		page.Dive.NextID = 0
	}

	renderTemplate(w, page)
}

func renderSite(w http.ResponseWriter, r *http.Request, divelog *DiveLog) {
	siteID := utils.ConvertAndCheckID(r.PathValue("id"), divelog.LargestSiteID())
	if siteID == 0 {
		renderNotFound(w, "site not found")
		return
	}
	site := divelog.DiveSites[siteID]

	renderTemplate(w, Page{
		Title:      site.Name,
		Supertitle: site.Region,
		Site:       NewSiteFull(site, divelog.Dives[1:]),
	})
}

func renderTags(w http.ResponseWriter, r *http.Request, divelog *DiveLog) {
	tags := make(map[string]int)
	for _, dive := range divelog.Dives[1:] {
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

func renderTaggedDives(w http.ResponseWriter, r *http.Request, divelog *DiveLog) {
	tag := r.PathValue("tag")
	dives := []*DiveHead{}
	for i := len(divelog.Dives) - 1; i > 0; i-- {
		dive := divelog.Dives[i]
		for _, t := range dive.Tags {
			if t == tag {
				dives = append(
					dives,
					NewDiveHead(dive, divelog.DiveSites[dive.DiveSiteID]),
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
	if err := _page_template.Execute(w, p); err != nil {
		trace(_error, "http: render template: %v", err)
	}
}
