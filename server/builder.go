package server

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"src.acicovic.me/divelog/server/utils"
	"src.acicovic.me/divelog/subsurface"
)

var bluefin DiveLog

func buildDatabase() {
	file, err := os.Open(bluefin.Metadata.Source)
	if err != nil {
		panic(fmt.Errorf("failed to open file %s: %v", bluefin.Metadata.Source, err))
	}
	defer file.Close()

	if err = subsurface.DecodeSubsurfaceDatabase(file, &SubsurfaceCallbackHandler{}); err != nil {
		panic(fmt.Errorf("failed to decode database in %s: %v", bluefin.Metadata.Source, err))
	}
}

type SubsurfaceCallbackHandler struct {
	lastSiteID int
	lastTripID int
	lastDiveID int
}

func (p *SubsurfaceCallbackHandler) HandleBegin() {
	bluefin.DiveSites = make([]*DiveSite, 1, 100)
	bluefin.DiveTrips = make([]*DiveTrip, 1, 100)
	bluefin.Dives = make([]*Dive, 1, 100)
	bluefin.sourceToSystemID = make(map[string]int)
}

func (p *SubsurfaceCallbackHandler) HandleDive(ddh subsurface.DiveDataHolder) int {
	regularTags := make([]string, 0, len(ddh.Tags))
	specialTags := make([]string, 0)
	for _, tag := range ddh.Tags {
		if utils.IsSpecialTag(tag) {
			specialTags = append(specialTags, tag)
		} else {
			regularTags = append(regularTags, tag)
		}
	}

	dive := &Dive{
		ID:     p.lastDiveID + 1,
		Number: ddh.DiveNumber,

		Duration:        ddh.Duration,
		Rating5:         ddh.Rating,
		Visibility5:     ddh.Visibility,
		Tags:            regularTags,
		Salinity:        ddh.WaterSalinity,
		DateTimeIn:      ddh.DateTime.Format(time.RFC3339),
		OperatorDM:      ddh.DiveMasterOrOperator,
		Buddy:           ddh.Buddy,
		Notes:           ddh.Notes,
		Suit:            ddh.Suit,
		CylSize:         ddh.CylinderSize,
		CylType:         ddh.CylinderDescription,
		StartPressure:   ddh.CylinderStartPressure,
		EndPressure:     ddh.CylinderEndPressure,
		Gas:             ddh.CylinderGas,
		Weights:         ddh.Weight,
		WeightsType:     ddh.WeightType,
		DCModel:         ddh.DiveComputerModel,
		DepthMax:        ddh.DepthMax,
		DepthMean:       ddh.DepthMean,
		TempWaterMin:    ddh.TemperatureWaterMin,
		TempAir:         ddh.TemperatureAir,
		SurfacePressure: ddh.SurfacePressure,

		datetime: ddh.DateTime,
	}
	trace(_build, "%v", dive)
	assert(dive.ID == len(bluefin.Dives), "invalid Dive.ID")

	siteID, ok := bluefin.sourceToSystemID[ddh.DiveSiteUUID]
	assert(ok, "DiveDataHolder.DiveSiteUUID is not mapped to DiveSite.ID")
	dive.DiveSiteID = siteID
	assert(siteID > 0 && siteID < len(bluefin.DiveSites), "invalid dive site ID mapping")
	assert(bluefin.DiveSites[siteID] != nil, "DiveSite ptr is nil")
	trace(_link, "%v -> %v", dive, bluefin.DiveSites[siteID])

	dive.DiveTripID = ddh.DiveTripID
	assert(ddh.DiveTripID > 0 && ddh.DiveTripID < len(bluefin.DiveTrips), "invalid dive trip ID")
	assert(bluefin.DiveTrips[ddh.DiveTripID] != nil, "DiveTrip ptr is nil")
	trace(_link, "%v -> %v", dive, bluefin.DiveTrips[ddh.DiveTripID])

	dive.ProcessSpecialTags(specialTags)
	dive.Normalize()

	bluefin.Dives = append(bluefin.Dives, dive)
	p.lastDiveID++

	return dive.ID
}

func (p *SubsurfaceCallbackHandler) HandleDiveSite(uuid string, name string, coords string, description string) int {
	region := UnlabeledRegion
	if strings.HasPrefix(description, PrefixForTagsInDescription) {
		var specialTags string
		if i := strings.IndexFunc(description, unicode.IsSpace); i != -1 {
			specialTags = strings.TrimPrefix(description[:i], PrefixForTagsInDescription)
			description = strings.TrimSpace(description[i:])
		} else {
			specialTags = strings.TrimPrefix(description, PrefixForTagsInDescription)
			description = ""
		}

		// DEVNOTE: DiveSite only supports one special tag for now: {RegionTagPrefix}{value}.
		// If there arises a need for more, this will need to be refactored.
		if after, ok := strings.CutPrefix(specialTags, RegionTagPrefix); ok {
			if value, ok := SpecialTagValueMappings[after]; ok {
				region = value
			}
		}
	}

	if strings.TrimSpace(description) == "" {
		description = UndefinedDescription
	}

	site := &DiveSite{
		ID:          p.lastSiteID + 1,
		Name:        name,
		Coordinates: coords,
		Description: description,
		Region:      region,

		sourceID: uuid,
	}
	trace(_build, "%v", site)
	assert(site.ID == len(bluefin.DiveSites), "invalid DiveSite.ID")

	bluefin.sourceToSystemID[site.sourceID] = site.ID
	trace(_map, "sourceToSystemID %q -> %d", site.sourceID, site.ID)

	bluefin.DiveSites = append(bluefin.DiveSites, site)
	p.lastSiteID++

	return site.ID
}

func (p *SubsurfaceCallbackHandler) HandleDiveTrip(label string) int {
	trip := &DiveTrip{
		ID:    p.lastTripID + 1,
		Label: label,
	}
	trace(_build, "%v", trip)
	assert(trip.ID == len(bluefin.DiveTrips), "invalid DiveTrip.ID")

	bluefin.DiveTrips = append(bluefin.DiveTrips, trip)
	p.lastTripID++

	return trip.ID
}

func (p *SubsurfaceCallbackHandler) HandleEnd() {
	assert(len(bluefin.Dives)-1 == p.lastDiveID, "invalid Dives slice length")
	assert(len(bluefin.DiveSites)-1 == p.lastSiteID, "invalid DiveSites slice length")
	assert(len(bluefin.DiveTrips)-1 == p.lastTripID, "invalid DiveTrips slice length")
}

func (p *SubsurfaceCallbackHandler) HandleGeoData(siteID int, cat int, label string) {
	assert(bluefin.DiveSites[siteID] != nil, "DiveSite ptr is nil")
	site := bluefin.DiveSites[siteID]
	for _, lbl := range site.GeoLabels {
		if lbl == label {
			return
		}
	}
	site.GeoLabels = append(site.GeoLabels, label)
}

func (p *SubsurfaceCallbackHandler) HandleHeader(program string, version string) {
	bluefin.Metadata.Program = program
	bluefin.Metadata.ProgramVersion = version
	bluefin.Metadata.Units = "metric" // DEVNOTE: make configurable?
}

func (p *SubsurfaceCallbackHandler) HandleSkip(element string) {
	// do nothing
}
