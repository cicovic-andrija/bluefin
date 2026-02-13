package server

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"src.acicovic.me/divelog/server/utils"
	"src.acicovic.me/divelog/subsurface"
)

func runAndWaitForBuilder() {
	errChannel := make(chan error)
	go builder(errChannel)

	select {
	case err := <-errChannel:
		if err != nil {
			panic(err)
		}
	case <-time.After(30 * time.Second):
		panic(errors.New("database initialization timed out"))
	}

	trace(_control, "mandatory database initialization on boot completed")
}

// Run in a goroutine.
// For simplicity, the goroutine will not be gracefully stopped,
// it will be force-stopped once the whole process is killed.
func builder(firstRun chan error) {
	var once sync.Once

	for {
		err := buildAndSwap()

		once.Do(func() {
			firstRun <- err
		})

		if err != nil {
			trace(_error, "divelog build failed: %v", err)
		} else {
			trace(_build, "divelog build completed successfully")
		}

		time.Sleep(time.Minute)
	}
}

func buildAndSwap() error {
	filePath, modTime, err := findLatestDataFile()
	if err != nil {
		return err
	}

	latestData := acquireDataAccess()
	divelog := &DiveLog{}
	if latestData != nil && !modTime.After(latestData.Metadata.modTime) {
		trace(_build, "builder found no newer data files, waiting for next iteration...")
		return nil
	}

	trace(_build, "divelog build started, from source file %s[mt:%s]", filePath, modTime)
	divelog.Metadata.Source = filePath
	divelog.Metadata.modTime = modTime
	divelog.Metadata.ModificationTime = modTime.Format(time.RFC3339)
	divelog.Metadata.Units = "metric"

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	if err = subsurface.DecodeSubsurfaceDatabase(file, to(divelog)); err != nil {
		return fmt.Errorf("failed to decode database: %v", err)
	}

	swapLatestData(divelog)

	return nil
}

func to(divelog *DiveLog) *SubsurfaceCallbackHandler {
	return &SubsurfaceCallbackHandler{
		divelog: divelog,
	}
}

type SubsurfaceCallbackHandler struct {
	divelog    *DiveLog
	lastSiteID int
	lastTripID int
	lastDiveID int
}

func (p *SubsurfaceCallbackHandler) HandleBegin() {
	p.divelog.DiveSites = make([]*DiveSite, 1, 100)
	p.divelog.DiveTrips = make([]*DiveTrip, 1, 100)
	p.divelog.Dives = make([]*Dive, 1, 100)
	p.divelog.sourceToSystemID = make(map[string]int)
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
	assert(dive.ID == len(p.divelog.Dives), "invalid Dive.ID")

	siteID, ok := p.divelog.sourceToSystemID[ddh.DiveSiteUUID]
	assert(ok, "DiveDataHolder.DiveSiteUUID is not mapped to DiveSite.ID")
	dive.DiveSiteID = siteID
	assert(siteID > 0 && siteID < len(p.divelog.DiveSites), "invalid dive site ID mapping")
	assert(p.divelog.DiveSites[siteID] != nil, "DiveSite ptr is nil")
	trace(_link, "%v -> %v", dive, p.divelog.DiveSites[siteID])

	dive.DiveTripID = ddh.DiveTripID
	assert(ddh.DiveTripID > 0 && ddh.DiveTripID < len(p.divelog.DiveTrips), "invalid dive trip ID")
	assert(p.divelog.DiveTrips[ddh.DiveTripID] != nil, "DiveTrip ptr is nil")
	trace(_link, "%v -> %v", dive, p.divelog.DiveTrips[ddh.DiveTripID])

	dive.ProcessSpecialTags(specialTags)
	dive.Normalize()

	p.divelog.Dives = append(p.divelog.Dives, dive)
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
	assert(site.ID == len(p.divelog.DiveSites), "invalid DiveSite.ID")

	p.divelog.sourceToSystemID[site.sourceID] = site.ID
	trace(_map, "sourceToSystemID %q -> %d", site.sourceID, site.ID)

	p.divelog.DiveSites = append(p.divelog.DiveSites, site)
	p.lastSiteID++

	return site.ID
}

func (p *SubsurfaceCallbackHandler) HandleDiveTrip(label string) int {
	trip := &DiveTrip{
		ID:    p.lastTripID + 1,
		Label: label,
	}
	trace(_build, "%v", trip)
	assert(trip.ID == len(p.divelog.DiveTrips), "invalid DiveTrip.ID")

	p.divelog.DiveTrips = append(p.divelog.DiveTrips, trip)
	p.lastTripID++

	return trip.ID
}

func (p *SubsurfaceCallbackHandler) HandleEnd() {
	assert(len(p.divelog.Dives)-1 == p.lastDiveID, "invalid Dives slice length")
	assert(len(p.divelog.DiveSites)-1 == p.lastSiteID, "invalid DiveSites slice length")
	assert(len(p.divelog.DiveTrips)-1 == p.lastTripID, "invalid DiveTrips slice length")
}

func (p *SubsurfaceCallbackHandler) HandleGeoData(siteID int, cat int, label string) {
	assert(p.divelog.DiveSites[siteID] != nil, "DiveSite ptr is nil")
	site := p.divelog.DiveSites[siteID]
	for _, lbl := range site.GeoLabels {
		if lbl == label {
			return
		}
	}
	site.GeoLabels = append(site.GeoLabels, label)
}

func (p *SubsurfaceCallbackHandler) HandleHeader(program string, version string) {
	p.divelog.Metadata.Program = program
	p.divelog.Metadata.ProgramVersion = version
}

func (p *SubsurfaceCallbackHandler) HandleSkip(element string) {
	// do nothing
}

func findLatestDataFile() (path string, mt time.Time, err error) {
	directoryPath := _control_block.watchDirectoryPath
	entries, err := os.ReadDir(directoryPath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, SubsurfaceDataFilePrefix) {
			continue
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			err = infoErr
			return
		}

		modTime := info.ModTime()
		if path == "" || modTime.After(mt) {
			mt = modTime
			path = filepath.Join(directoryPath, name)
		}
	}

	if path == "" {
		err = fmt.Errorf(
			"no files with prefix %q found in %s",
			SubsurfaceDataFilePrefix,
			directoryPath,
		)
	}

	return
}
