package server

import (
	"fmt"
	"strings"
)

type All struct {
	DiveSites []*DiveSite `json:"dive_sites"`
	DiveTrips []*DiveTrip `json:"dive_trips"`
	Dives     []*Dive     `json:"dives"`
}

type SiteHead struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type SiteFull struct {
	*DiveSite
	LinkedDives []*DiveHead `json:"linked_dives"`
}

type DiveHead struct {
	ID               int    `json:"id"`
	ShortLabel       string `json:"short_label"`
	DateTimeInPretty string `json:"date_time_in_pretty"`
	Award            string `json:"award,omitempty"`
}

type DiveFull struct {
	*Dive
	DiveSiteName     string `json:"dive_site_name"`
	DateTimeInPretty string `json:"date_time_in_pretty"`
	NextID           int    `json:"-"`
	PrevID           int    `json:"-"`
}

type Trip struct {
	ID          int         `json:"id"`
	Label       string      `json:"label"`
	LinkedDives []*DiveHead `json:"linked_dives"`
}

type GroupedSites struct {
	Region      string
	LinkedSites []*SiteHead
}

func NewDiveHead(dive *Dive, diveSite *DiveSite) *DiveHead {
	return &DiveHead{
		ID:               dive.ID,
		ShortLabel:       fmt.Sprintf("Dive %d: %s", dive.Number, diveSite.ShortName()),
		DateTimeInPretty: dive.datetime.Format("January 2 2006, 15:04"),
		Award:            dive.Award,
	}
}

func NewDiveFull(dive *Dive, diveSite *DiveSite) *DiveFull {
	return &DiveFull{
		Dive:             dive,
		DiveSiteName:     diveSite.Name,
		DateTimeInPretty: dive.datetime.Format("January 2 2006, 15:04"),
		NextID:           dive.ID + 1,
		PrevID:           dive.ID - 1,
	}
}

func NewSiteFull(site *DiveSite, allDives []*Dive) *SiteFull {
	s := &SiteFull{DiveSite: site}
	for _, dive := range allDives {
		if dive.DiveSiteID == site.ID {
			s.LinkedDives = append(s.LinkedDives, NewDiveHead(dive, site))
		}
	}
	return s
}

func (s *SiteFull) URLLongLat() string {
	return strings.Replace(s.Coordinates, " ", ",", 1)
}

type Page struct {
	Title        string
	Supertitle   string
	Trips        []*Trip
	GroupedSites []*GroupedSites
	Dives        []*DiveHead
	Tags         map[string]int
	Dive         *DiveFull
	Site         *SiteFull
	About        bool
	NotFound     bool
}

func (p *Page) check() bool {
	c := 0
	if p.Trips != nil {
		c++
	}
	if p.GroupedSites != nil {
		c++
	}
	if p.Dives != nil {
		c++
	}
	if p.Tags != nil {
		c++
	}
	if p.Dive != nil {
		c++
	}
	if p.Site != nil {
		c++
	}
	if p.About {
		c++
	}
	if p.NotFound {
		c++
	}
	return c == 1
}
