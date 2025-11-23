package subsurface

import (
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
)

// WIP
// Not really, but the decoder is tested only on my own data files.

const (
	IntNull = 0
)

var (
	ErrNilReader     = errors.New("io.Reader is nil")
	ErrNilHandler    = errors.New("subsurface.Handler is nil")
	ErrInvalidFormat = errors.New("XML database is not in the valid format")
)

type Handler interface {
	HandleBegin()
	HandleEnd()
	HandleHeader(program string, version string)
	HandleSkip(element string)
	HandleDiveSite(uuid string, name string, coords string, description string) int
	HandleGeoData(siteID int, cat int, label string)
	HandleDiveTrip(label string) int
	HandleDive(ddh DiveDataHolder) int
}

type Decoder struct {
	XMLDecoder *xml.Decoder
}

type DiveDataHolder struct {
	DiveNumber            int
	DiveTripID            int
	DiveSiteUUID          string
	Rating                int
	Visibility            int
	SAC                   string
	Tags                  []string
	WaterSalinity         string
	DateTime              time.Time
	Duration              string
	DiveMasterOrOperator  string
	Buddy                 string
	Notes                 string
	Suit                  string
	CylinderSize          string
	CylinderWorkPressure  string
	CylinderDescription   string
	CylinderStartPressure string
	CylinderEndPressure   string
	CylinderGas           string
	Weight                string
	WeightType            string
	DiveComputerModel     string
	DiveComputerDeviceID  string
	DiveComputerDiveID    string
	DepthMax              string
	DepthMean             string
	TemperatureWaterMin   string
	TemperatureAir        string
	SurfacePressure       string
}

func DecodeSubsurfaceDatabase(r io.Reader, h Handler) error {
	if r == nil {
		return ErrNilReader
	}
	if h == nil {
		return ErrNilHandler
	}

	var (
		decoder = &Decoder{
			XMLDecoder: xml.NewDecoder(r),
		}
		startTag *xml.StartElement
		err      error
	)

	// <divelog ...>
	if startTag, err = decoder.ExpectStart("divelog"); err != nil {
		return err
	}
	h.HandleBegin()

	program, _ := FindAttribute(startTag, "program")
	version, _ := FindAttribute(startTag, "version")
	h.HandleHeader(program, version)

	// <settings>
	if err = decoder.SkipElement("settings"); err != nil {
		return err
	}
	h.HandleSkip("settings")
	// </settings>

	// <divesites>
	if _, err = decoder.ExpectStart("divesites"); err != nil {
		// DEVNOTE: the database may contain no dive site entries, but this
		// decoder will report it as a format error
		return err
	}

	for {
		if startTag, err = decoder.NextOrEnd("site", "divesites"); err != nil {
			return err
		}
		if startTag != nil {
			// <site ...> 1..N
			if siteXML, err := DecodeSiteXML(decoder, startTag); err != nil {
				return err
			} else {
				// DEVNOTE: this could have been parsed the same way the dive data was parsed
				// it was left like this to demonstrate how powerful Go's XML parser can be
				siteID := h.HandleDiveSite(siteXML.UUID, siteXML.Name, siteXML.GPS, siteXML.Description)
				for _, geoData := range siteXML.Geos {
					if cat, err := strconv.Atoi(geoData.Cat); err != nil {
						return ErrInvalidFormat
					} else {
						h.HandleGeoData(siteID, cat, geoData.Value)
					}
				}
			}
			// </site>
		} else {
			// </divesites>
			break
		}
	}

	// <dives>
	if _, err = decoder.ExpectStart("dives"); err != nil {
		// DEVNOTE: the database may contain no dive entries, or only dive entries,
		// or combined trip and dive entries, but this decoder will report it as a format error
		return err
	}

	for {
		if startTag, err = decoder.NextOrEnd("trip", "dives"); err != nil {
			return err
		}
		if startTag != nil {
			// <trip ...> 1..N
			location, _ := FindAttribute(startTag, "location")
			tripID := h.HandleDiveTrip(location)
			for {
				if startTag, err = decoder.NextOrEnd("dive", "trip"); err != nil {
					return err
				}
				if startTag != nil {
					// <dive ...> 1..N
					if diveXML, err := DecodeDiveXML(decoder, startTag); err != nil {
						return err
					} else {
						if err := FlattenAndReport(diveXML, tripID, h); err != nil {
							return err
						}
					}
				} else {
					// </trip>
					break
				}
			}
		} else {
			// </dives>
			break
		}
	}

	h.HandleEnd()
	// </divelog>

	return nil
}

func FlattenAndReport(diveXML *DiveXML, tripID int, h Handler) error {
	var (
		ddh = DiveDataHolder{
			DiveTripID:            tripID,
			DiveSiteUUID:          diveXML.DiveSiteUUID,
			SAC:                   diveXML.SAC,
			WaterSalinity:         diveXML.WaterSalinity,
			Duration:              diveXML.Duration,
			DiveMasterOrOperator:  diveXML.DiveMaster,
			Buddy:                 diveXML.Buddy,
			Notes:                 diveXML.Notes,
			Suit:                  diveXML.Suit,
			CylinderSize:          diveXML.Cylinder.Size,
			CylinderWorkPressure:  diveXML.Cylinder.WorkPressure,
			CylinderDescription:   diveXML.Cylinder.Description,
			CylinderStartPressure: diveXML.Cylinder.Start,
			CylinderEndPressure:   diveXML.Cylinder.End,
			CylinderGas:           diveXML.Cylinder.O2,
			Weight:                diveXML.WeightSystem.Weight,
			WeightType:            diveXML.WeightSystem.Description,
			DiveComputerModel:     diveXML.DiveComputer.Model,
			DiveComputerDeviceID:  diveXML.DiveComputer.DeviceID,
			DiveComputerDiveID:    diveXML.DiveComputer.DiveID,
			DepthMax:              diveXML.DiveComputer.DepthInfo.Max,
			DepthMean:             diveXML.DiveComputer.DepthInfo.Mean,
			TemperatureAir:        diveXML.TemperatureManual.Air,
			SurfacePressure:       diveXML.DiveComputer.SurfaceInfo.Pressure,
		}
		err error
	)

	if diveXML.Number == "" {
		ddh.DiveNumber = IntNull
	} else if ddh.DiveNumber, err = strconv.Atoi(diveXML.Number); err != nil {
		return ErrInvalidFormat
	}

	if diveXML.Rating != "" {
		if ddh.Rating, err = strconv.Atoi(diveXML.Rating); err != nil {
			return ErrInvalidFormat
		}
	} else {
		ddh.Rating = IntNull
	}

	if diveXML.Visibility != "" {
		if ddh.Visibility, err = strconv.Atoi(diveXML.Visibility); err != nil {
			return ErrInvalidFormat
		}
	} else {
		ddh.Visibility = IntNull
	}

	for _, tag := range strings.Split(diveXML.Tags, ",") {
		if trimmed := strings.TrimSpace(tag); trimmed != "" {
			ddh.Tags = append(ddh.Tags, trimmed)
		}
	}

	// date format is yyyy-mm-dd
	// time format is hh:mm:ss
	if diveXML.Date == "" {
		ddh.DateTime = time.Time{}
	} else {
		dateTimeStr := diveXML.Date + "T" + diveXML.Time + "Z"
		if diveXML.Time == "" { // unlikely
			dateTimeStr = diveXML.Date + "T00:00:00Z"
		}
		if ddh.DateTime, err = time.Parse(time.RFC3339, dateTimeStr); err != nil {
			return ErrInvalidFormat
		}
	}

	if diveXML.DiveComputer.TemperatureInfo.WaterMin != "" {
		ddh.TemperatureWaterMin = diveXML.DiveComputer.TemperatureInfo.WaterMin
	} else {
		ddh.TemperatureWaterMin = diveXML.TemperatureManual.Water
	}

	h.HandleDive(ddh)
	return nil
}

func DecodeSiteXML(decoder *Decoder, tok *xml.StartElement) (*SiteXML, error) {
	siteXML := &SiteXML{}
	err := decoder.XMLDecoder.DecodeElement(siteXML, tok)
	return siteXML, err
}

func DecodeDiveXML(decoder *Decoder, tok *xml.StartElement) (*DiveXML, error) {
	diveXML := &DiveXML{}
	err := decoder.XMLDecoder.DecodeElement(diveXML, tok)
	return diveXML, err
}

func (d *Decoder) Token() (tok xml.Token, err error) {
	for {
		tok, err = d.XMLDecoder.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.CharData:
			if len(strings.TrimSpace(string(t))) > 0 {
				return
			}
		default:
			return
		}
	}
}

func (d *Decoder) ExpectStart(tag string) (*xml.StartElement, error) {
	tok, err := d.Token()
	if err != nil {
		return nil, ErrInvalidFormat
	}
	switch tok := tok.(type) {
	case xml.StartElement:
		if tok.Name.Local == tag {
			return &tok, nil
		}
	}
	return nil, ErrInvalidFormat
}

func (d *Decoder) ExpectAnyStart() (*xml.StartElement, error) {
	tok, err := d.Token()
	if err != nil {
		return nil, ErrInvalidFormat
	}
	switch tok := tok.(type) {
	case xml.StartElement:
		return &tok, nil
	}
	return nil, ErrInvalidFormat
}

func (d *Decoder) ExpectEnd(tag string) (*xml.EndElement, error) {
	tok, err := d.Token()
	if err != nil {
		return nil, ErrInvalidFormat
	}
	switch tok := tok.(type) {
	case xml.EndElement:
		if tok.Name.Local == tag {
			return &tok, nil
		}
	}
	return nil, ErrInvalidFormat
}

func (d *Decoder) NextOrEnd(next string, end string) (*xml.StartElement, error) {
	tok, err := d.Token()
	if err != nil {
		return nil, ErrInvalidFormat
	}
	switch tok := tok.(type) {
	case xml.StartElement:
		if tok.Name.Local == next {
			return &tok, nil
		}
	case xml.EndElement:
		if tok.Name.Local == end {
			return nil, nil
		}
	}
	return nil, ErrInvalidFormat
}

func (d *Decoder) SkipElement(tag string) error {
	if _, err := d.ExpectStart(tag); err != nil {
		return err
	}
	return d.XMLDecoder.Skip()
}

func FindAttribute(tok *xml.StartElement, name string) (val string, ok bool) {
	for _, attr := range tok.Attr {
		if attr.Name.Local == name {
			ok = true
			val = attr.Value
			return
		}
	}
	return
}

func IsValidDateTime(t time.Time) bool {
	return !t.IsZero()
}
