package server

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
		return fmt.Errorf("failed to decode database: %w", err)
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

func (p *SubsurfaceCallbackHandler) HandleBegin() error {
	p.divelog.DiveSites = make([]*DiveSite, 1, 100)
	p.divelog.DiveTrips = make([]*DiveTrip, 1, 100)
	p.divelog.Dives = make([]*Dive, 1, 100)
	p.divelog.sourceToSystemID = make(map[string]int)
	return nil
}

func (p *SubsurfaceCallbackHandler) HandleDive(ddh subsurface.DiveDataHolder) (int, error) {
	// Pure loader: assign raw tags without processing
	dive := &Dive{
		ID:     p.lastDiveID + 1,
		Number: ddh.DiveNumber,

		Duration:        ddh.Duration,
		Rating5:         ddh.Rating,
		Visibility5:     ddh.Visibility,
		Tags:            ddh.Tags, // Raw tags - normalization happens later
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
	if dive.ID != len(p.divelog.Dives) {
		return 0, fmt.Errorf("invalid Dive.ID: got %d, want %d", dive.ID, len(p.divelog.Dives))
	}

	siteID, ok := p.divelog.sourceToSystemID[ddh.DiveSiteUUID]
	if !ok {
		return 0, fmt.Errorf("DiveDataHolder.DiveSiteUUID=%q is not mapped to DiveSite.ID", ddh.DiveSiteUUID)
	}
	dive.DiveSiteID = siteID
	if siteID <= 0 || siteID >= len(p.divelog.DiveSites) {
		return 0, fmt.Errorf("invalid dive site ID mapping: siteID=%d, sitesLen=%d", siteID, len(p.divelog.DiveSites))
	}
	if p.divelog.DiveSites[siteID] == nil {
		return 0, fmt.Errorf("DiveSite ptr is nil for siteID=%d", siteID)
	}
	trace(_link, "%v -> %v", dive, p.divelog.DiveSites[siteID])

	dive.DiveTripID = ddh.DiveTripID
	if ddh.DiveTripID <= 0 || ddh.DiveTripID >= len(p.divelog.DiveTrips) {
		return 0, fmt.Errorf("invalid dive trip ID: tripID=%d, tripsLen=%d", ddh.DiveTripID, len(p.divelog.DiveTrips))
	}
	if p.divelog.DiveTrips[ddh.DiveTripID] == nil {
		return 0, fmt.Errorf("DiveTrip ptr is nil for tripID=%d", ddh.DiveTripID)
	}
	trace(_link, "%v -> %v", dive, p.divelog.DiveTrips[ddh.DiveTripID])

	// No normalization here - that happens in DiveLog.Normalize()

	p.divelog.Dives = append(p.divelog.Dives, dive)
	p.lastDiveID++

	return dive.ID, nil
}

func (p *SubsurfaceCallbackHandler) HandleDiveSite(uuid string, name string, coords string, description string) (int, error) {
	// Pure loader: store raw description, leave Region empty for normalization later
	site := &DiveSite{
		ID:          p.lastSiteID + 1,
		Name:        name,
		Coordinates: coords,
		Description: description, // Raw description - normalization happens later
		Region:      "",          // Empty - will be set in DiveLog.Normalize()

		sourceID: uuid,
	}
	trace(_build, "%v", site)
	if site.ID != len(p.divelog.DiveSites) {
		return 0, fmt.Errorf("invalid DiveSite.ID: got %d, want %d", site.ID, len(p.divelog.DiveSites))
	}

	p.divelog.sourceToSystemID[site.sourceID] = site.ID
	trace(_map, "sourceToSystemID %q -> %d", site.sourceID, site.ID)

	p.divelog.DiveSites = append(p.divelog.DiveSites, site)
	p.lastSiteID++

	return site.ID, nil
}

func (p *SubsurfaceCallbackHandler) HandleDiveTrip(label string) (int, error) {
	trip := &DiveTrip{
		ID:    p.lastTripID + 1,
		Label: label,
	}
	trace(_build, "%v", trip)
	if trip.ID != len(p.divelog.DiveTrips) {
		return 0, fmt.Errorf("invalid DiveTrip.ID: got %d, want %d", trip.ID, len(p.divelog.DiveTrips))
	}

	p.divelog.DiveTrips = append(p.divelog.DiveTrips, trip)
	p.lastTripID++

	return trip.ID, nil
}

func (p *SubsurfaceCallbackHandler) HandleEnd() error {
	// All normalization and validation happens here
	return p.divelog.Normalize()
}

func (p *SubsurfaceCallbackHandler) HandleGeoData(siteID int, cat int, label string) error {
	if siteID < 0 || siteID >= len(p.divelog.DiveSites) {
		return fmt.Errorf("invalid siteID=%d for DiveSites len=%d", siteID, len(p.divelog.DiveSites))
	}
	if p.divelog.DiveSites[siteID] == nil {
		return fmt.Errorf("DiveSite ptr is nil for siteID=%d", siteID)
	}
	// Pure loader: append raw label without deduplication - normalization happens later
	site := p.divelog.DiveSites[siteID]
	site.GeoLabels = append(site.GeoLabels, label)
	return nil
}

func (p *SubsurfaceCallbackHandler) HandleHeader(program string, version string) error {
	p.divelog.Metadata.Program = program
	p.divelog.Metadata.ProgramVersion = version
	return nil
}

func (p *SubsurfaceCallbackHandler) HandleSkip(element string) error {
	// do nothing
	return nil
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
