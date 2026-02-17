package server

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"src.acicovic.me/divelog/server/utils"
)

type DiveLog struct {
	Metadata  DiveLogMetadata
	DiveSites []*DiveSite
	DiveTrips []*DiveTrip
	Dives     []*Dive

	sourceToSystemID map[string]int
}

type DiveLogMetadata struct {
	Program          string `json:"program"`
	ProgramVersion   string `json:"program_version"`
	Source           string `json:"source"`
	ModificationTime string `json:"modification_time"`
	Units            string `json:"units"`

	modTime time.Time
}

type DiveSite struct {
	ID   int    `json:"id"`
	Name string `json:"name"`

	Coordinates string   `json:"coordinates,omitempty"`
	Description string   `json:"description,omitempty"`
	Region      string   `json:"region,omitempty"`
	GeoLabels   []string `json:"geo_labels,omitempty"`

	sourceID string
}

type DiveTrip struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

type Dive struct {
	ID         int `json:"id"`
	Number     int `json:"number"`
	DiveSiteID int `json:"dive_site_id"`
	DiveTripID int `json:"dive_trip_id"`

	Duration        string   `json:"duration,omitempty"`
	Rating5         int      `json:"rating5,omitempty"`
	Visibility5     int      `json:"visibility5,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	Salinity        string   `json:"salinity,omitempty"`
	DateTimeIn      string   `json:"date_time_in,omitempty"`
	OperatorDM      string   `json:"operator_dm,omitempty"`
	Buddy           string   `json:"buddy,omitempty"`
	Notes           string   `json:"notes,omitempty"`
	Suit            string   `json:"suit,omitempty"`
	CylSize         string   `json:"cyl_size,omitempty"`
	CylType         string   `json:"cyl_type,omitempty"`
	StartPressure   string   `json:"start_pressure,omitempty"`
	EndPressure     string   `json:"end_pressure,omitempty"`
	Gas             string   `json:"gas,omitempty"`
	Weights         string   `json:"weights,omitempty"`
	WeightsType     string   `json:"weights_type,omitempty"`
	DCModel         string   `json:"dc_model,omitempty"`
	DepthMax        string   `json:"depth_max,omitempty"`
	DepthMean       string   `json:"depth_mean,omitempty"`
	TempWaterMin    string   `json:"temp_water_min,omitempty"`
	TempAir         string   `json:"temp_air,omitempty"`
	SurfacePressure string   `json:"surface_pressure,omitempty"`
	Award           string   `json:"award,omitempty"`

	datetime time.Time
}

func (s *DiveSite) String() string {
	return fmt.Sprintf("S%d:[%s]", s.ID, s.Name)
}

func (s *DiveSite) ShortName() string {
	return strings.TrimSpace(strings.Split(s.Name, ",")[0])
}

func (s *DiveSite) FormattedCoordinates() string {
	parts := strings.Fields(strings.TrimSpace(s.Coordinates))
	return fmt.Sprintf("lat = %s, long = %s", parts[0], parts[1])
}

func (t *DiveTrip) String() string {
	return fmt.Sprintf("T%d:[%s]", t.ID, t.Label)
}

func (d *Dive) Ago() string {
	years, months, days := utils.DurationToYMD(d.datetime, time.Now().UTC())
	return fmt.Sprintf("%dy %dm %dd", years, months, days)
}

func (d *Dive) String() string {
	return fmt.Sprintf("D%d:[%s]", d.ID, d.datetime.Format(time.DateOnly))
}

func (d *Dive) Normalize() {
	if strings.HasPrefix(d.Salinity, "1000") {
		d.Salinity = "fresh water"
	} else if strings.HasPrefix(d.Salinity, "1030") {
		d.Salinity = "salt water"
	} else {
		d.Salinity = ""
	}

	if d.Gas == "" {
		d.Gas = "air"
	} else { // e.g. "32%"
		d.Gas = "nitrox " + d.Gas
	}

	if cylType, ok := CylinderTypeMappings[d.CylType]; ok {
		d.CylType = cylType
	} else {
		d.CylType = "unrecognized"
	}
}

func (d *Dive) IsTaggedWith(tag string) bool {
	if tag == "" {
		return true
	}
	for _, t := range d.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (d *Dive) ProcessSpecialTags(specialTags []string) {
	for _, tag := range specialTags {
		key, value := utils.ParseSpecialTag(tag)
		switch key {
		case "award":
			if mappedAward, ok := AwardMappings[value]; ok {
				d.Award = mappedAward
			}
		}
	}
}

func (dl *DiveLog) LargestDiveID() int {
	return len(dl.Dives) - 1
}

func (dl *DiveLog) LargestSiteID() int {
	return len(dl.DiveSites) - 1
}

// Normalize performs all normalization operations on the DiveLog data.
// This includes structural validation, tag processing, region extraction,
// and calling individual Dive normalization methods.
func (dl *DiveLog) Normalize() error {
	// Validate structural invariants: index == ID for all slices
	if err := dl.validateStructure(); err != nil {
		return err
	}

	// Normalize DiveSites
	for i := 1; i < len(dl.DiveSites); i++ {
		if err := dl.normalizeDiveSite(dl.DiveSites[i]); err != nil {
			return fmt.Errorf("normalize DiveSite[%d]: %w", i, err)
		}
	}

	// Normalize GeoLabels (deduplicate)
	for i := 1; i < len(dl.DiveSites); i++ {
		dl.normalizeGeoLabels(dl.DiveSites[i])
	}

	// Normalize Dives
	for i := 1; i < len(dl.Dives); i++ {
		if err := dl.normalizeDive(dl.Dives[i]); err != nil {
			return fmt.Errorf("normalize Dive[%d]: %w", i, err)
		}
	}

	return nil
}

func (dl *DiveLog) validateStructure() error {
	// Validate Dives
	for i := 1; i < len(dl.Dives); i++ {
		if dl.Dives[i] == nil {
			return fmt.Errorf("Dive[%d] is nil", i)
		}
		if dl.Dives[i].ID != i {
			return fmt.Errorf("invalid Dive.ID: got %d, want %d", dl.Dives[i].ID, i)
		}
	}

	// Validate DiveSites
	for i := 1; i < len(dl.DiveSites); i++ {
		if dl.DiveSites[i] == nil {
			return fmt.Errorf("DiveSite[%d] is nil", i)
		}
		if dl.DiveSites[i].ID != i {
			return fmt.Errorf("invalid DiveSite.ID: got %d, want %d", dl.DiveSites[i].ID, i)
		}
	}

	// Validate DiveTrips
	for i := 1; i < len(dl.DiveTrips); i++ {
		if dl.DiveTrips[i] == nil {
			return fmt.Errorf("DiveTrip[%d] is nil", i)
		}
		if dl.DiveTrips[i].ID != i {
			return fmt.Errorf("invalid DiveTrip.ID: got %d, want %d", dl.DiveTrips[i].ID, i)
		}
	}

	return nil
}

func (dl *DiveLog) normalizeDiveSite(site *DiveSite) error {
	if site == nil {
		return fmt.Errorf("site is nil")
	}

	description := site.Description
	region := UnlabeledRegion

	// Parse and strip leading PrefixForTagsInDescription metadata
	if strings.HasPrefix(description, PrefixForTagsInDescription) {
		var specialTags string
		if i := strings.IndexFunc(description, unicode.IsSpace); i != -1 {
			specialTags = strings.TrimPrefix(description[:i], PrefixForTagsInDescription)
			description = strings.TrimSpace(description[i:])
		} else {
			specialTags = strings.TrimPrefix(description, PrefixForTagsInDescription)
			description = ""
		}

		// Extract region from special tags
		if after, ok := strings.CutPrefix(specialTags, RegionTagPrefix); ok {
			if value, ok := SpecialTagValueMappings[after]; ok {
				region = value
			}
		}
	}

	// Set default description if empty
	if strings.TrimSpace(description) == "" {
		description = UndefinedDescription
	}

	site.Description = description
	site.Region = region

	return nil
}

func (dl *DiveLog) normalizeGeoLabels(site *DiveSite) {
	if site == nil || len(site.GeoLabels) == 0 {
		return
	}

	// Deduplicate labels
	seen := make(map[string]bool)
	deduped := make([]string, 0, len(site.GeoLabels))
	for _, label := range site.GeoLabels {
		trimmed := strings.TrimSpace(label)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			deduped = append(deduped, trimmed)
		}
	}
	site.GeoLabels = deduped
}

func (dl *DiveLog) normalizeDive(dive *Dive) error {
	if dive == nil {
		return fmt.Errorf("dive is nil")
	}

	// Clean tags: trim whitespace and drop empty entries
	cleanedTags := make([]string, 0, len(dive.Tags))
	for _, tag := range dive.Tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			cleanedTags = append(cleanedTags, trimmed)
		}
	}

	// Split tags into special (prefix _) vs regular
	regularTags := make([]string, 0, len(cleanedTags))
	specialTags := make([]string, 0)
	for _, tag := range cleanedTags {
		if utils.IsSpecialTag(tag) {
			specialTags = append(specialTags, tag)
		} else {
			regularTags = append(regularTags, tag)
		}
	}

	// Process special tags and normalize dive
	dive.ProcessSpecialTags(specialTags)
	dive.Tags = regularTags
	dive.Normalize()

	return nil
}
