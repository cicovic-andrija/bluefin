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

// Only one thread at a time, builder(), will ever access this pointer,
// so there is no need to guard it (to keep things simple for now).
var _divelog *DiveLog

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
		err := buildFromLatestDataFile()

		once.Do(func() {
			firstRun <- err
		})

		if err != nil {
			trace(_error, "database build failed: %v", err)
		}

		time.Sleep(time.Minute)
	}
}

func buildFromLatestDataFile() error {
	filePath, modTime, err := findLatestDataFile()
	if err != nil {
		return err
	}

	latestBuild := acquireDataAccess()
	if latestBuild == nil || modTime.After(latestBuild.Metadata.modTime) {
		_divelog = &DiveLog{}
		_divelog.Metadata.Source = filePath
		_divelog.Metadata.modTime = modTime
		_divelog.Metadata.ModificationTime = modTime.Format(time.RFC3339)
	} else {
		trace(_build, "builder found no newer data files, waiting for next iteration...")
		return nil
	}

	trace(_build, "database build started, from source file %s", filePath)
	if err := buildDatabase(); err != nil {
		return err
	}

	swapLatestData(_divelog)

	trace(_build, "database build completed with modification time %s", modTime)
	return nil
}

func buildDatabase() error {
	path := _divelog.Metadata.Source
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", path, err)
	}
	defer file.Close()

	if err = subsurface.DecodeSubsurfaceDatabase(file, &SubsurfaceCallbackHandler{}); err != nil {
		return fmt.Errorf("failed to decode database in %s: %v", path, err)
	}

	return nil
}

type SubsurfaceCallbackHandler struct {
	lastSiteID int
	lastTripID int
	lastDiveID int
}

func (p *SubsurfaceCallbackHandler) HandleBegin() {
	_divelog.DiveSites = make([]*DiveSite, 1, 100)
	_divelog.DiveTrips = make([]*DiveTrip, 1, 100)
	_divelog.Dives = make([]*Dive, 1, 100)
	_divelog.sourceToSystemID = make(map[string]int)
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
	assert(dive.ID == len(_divelog.Dives), "invalid Dive.ID")

	siteID, ok := _divelog.sourceToSystemID[ddh.DiveSiteUUID]
	assert(ok, "DiveDataHolder.DiveSiteUUID is not mapped to DiveSite.ID")
	dive.DiveSiteID = siteID
	assert(siteID > 0 && siteID < len(_divelog.DiveSites), "invalid dive site ID mapping")
	assert(_divelog.DiveSites[siteID] != nil, "DiveSite ptr is nil")
	trace(_link, "%v -> %v", dive, _divelog.DiveSites[siteID])

	dive.DiveTripID = ddh.DiveTripID
	assert(ddh.DiveTripID > 0 && ddh.DiveTripID < len(_divelog.DiveTrips), "invalid dive trip ID")
	assert(_divelog.DiveTrips[ddh.DiveTripID] != nil, "DiveTrip ptr is nil")
	trace(_link, "%v -> %v", dive, _divelog.DiveTrips[ddh.DiveTripID])

	dive.ProcessSpecialTags(specialTags)
	dive.Normalize()

	_divelog.Dives = append(_divelog.Dives, dive)
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
	assert(site.ID == len(_divelog.DiveSites), "invalid DiveSite.ID")

	_divelog.sourceToSystemID[site.sourceID] = site.ID
	trace(_map, "sourceToSystemID %q -> %d", site.sourceID, site.ID)

	_divelog.DiveSites = append(_divelog.DiveSites, site)
	p.lastSiteID++

	return site.ID
}

func (p *SubsurfaceCallbackHandler) HandleDiveTrip(label string) int {
	trip := &DiveTrip{
		ID:    p.lastTripID + 1,
		Label: label,
	}
	trace(_build, "%v", trip)
	assert(trip.ID == len(_divelog.DiveTrips), "invalid DiveTrip.ID")

	_divelog.DiveTrips = append(_divelog.DiveTrips, trip)
	p.lastTripID++

	return trip.ID
}

func (p *SubsurfaceCallbackHandler) HandleEnd() {
	assert(len(_divelog.Dives)-1 == p.lastDiveID, "invalid Dives slice length")
	assert(len(_divelog.DiveSites)-1 == p.lastSiteID, "invalid DiveSites slice length")
	assert(len(_divelog.DiveTrips)-1 == p.lastTripID, "invalid DiveTrips slice length")
}

func (p *SubsurfaceCallbackHandler) HandleGeoData(siteID int, cat int, label string) {
	assert(_divelog.DiveSites[siteID] != nil, "DiveSite ptr is nil")
	site := _divelog.DiveSites[siteID]
	for _, lbl := range site.GeoLabels {
		if lbl == label {
			return
		}
	}
	site.GeoLabels = append(site.GeoLabels, label)
}

func (p *SubsurfaceCallbackHandler) HandleHeader(program string, version string) {
	_divelog.Metadata.Program = program
	_divelog.Metadata.ProgramVersion = version
	_divelog.Metadata.Units = "metric"
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
