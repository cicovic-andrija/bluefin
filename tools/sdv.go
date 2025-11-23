package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"src.acicovic.me/divelog/subsurface"
)

// Subsurface Decoder Validator

func main() {
	if len(os.Args) != 2 {
		fmt.Println("provide file name as the first program argument")
		os.Exit(0x1)
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("failed to open file: %v\n", err)
		os.Exit(0x2)
	}
	defer file.Close()

	if err := subsurface.DecodeSubsurfaceDatabase(file, Handler{fname: os.Args[1]}); err != nil {
		fmt.Printf("decoding error: %v\n", err)
		os.Exit(0x3)
	}
}

type Handler struct {
	fname string
}

func (h Handler) HandleBegin() {
	fmt.Printf("SUBSURFACE_DATABASE %q\n", filepath.Base(h.fname))
}

func (h Handler) HandleEnd() {
	fmt.Printf("END.\n")
}

func (h Handler) HandleSkip(element string) {
	fmt.Printf(">>>>>\nSKIP ELEMENT %q\n<<<<<\n", element)
}

func (h Handler) HandleHeader(program string, version string) {
	fmt.Printf("\tHEADER\n")
	fmt.Printf("\t\tPROGRAM = %q\n", program)
	fmt.Printf("\t\tVERSION = %q\n", version)

}

func (h Handler) HandleDiveSite(uuid string, name string, coords string, description string) int {
	fmt.Printf("\tDIVE_SITE\n")
	fmt.Printf("\t\tUUID = %q\n\t\tNAME = %q\n\t\tCOORDS = %q\n\t\tDESCRIPTION = %q\n", uuid, name, coords, description)
	return 0
}

func (h Handler) HandleGeoData(id int, cat int, label string) {
	fmt.Printf("\t\tGEO_DATA\n")
	fmt.Printf("\t\t\tCATEGORY = %d\n\t\t\tLABEL = %q\n", cat, label)
}

func (h Handler) HandleDiveTrip(label string) int {
	fmt.Printf("\tDIVE_TRIP %q\n", label)
	return 0
}

func (h Handler) HandleDive(ddh subsurface.DiveDataHolder) int {
	fmt.Printf("\t\tDIVE\n")
	fmt.Printf("\t\t\tNUMBER = %d\n", ddh.DiveNumber)
	fmt.Printf("\t\t\tRATING = %d\n", ddh.Rating)
	fmt.Printf("\t\t\tVISIBILITY = %d\n", ddh.Visibility)
	fmt.Printf("\t\t\tSAC = %q\n", ddh.SAC)
	if len(ddh.Tags) > 0 {
		fmt.Printf("\t\t\tTAGS\n")
		for _, tag := range ddh.Tags {
			fmt.Printf("\t\t\t\t%q\n", tag)
		}
	}
	fmt.Printf("\t\t\tWATER_SALINITY = %q\n", ddh.WaterSalinity)
	fmt.Printf("\t\t\tDATE_TIME = %s\n", ddh.DateTime.Format(time.RFC1123Z))
	fmt.Printf("\t\t\tDURATION = %q\n", ddh.Duration)
	fmt.Printf("\t\t\tDIVE_OPERATOR = %q\n", ddh.DiveMasterOrOperator)
	fmt.Printf("\t\t\tBUDDY = %q\n", ddh.Buddy)
	fmt.Printf("\t\t\tNOTES = %q\n", ddh.Notes)
	fmt.Printf("\t\t\tSUIT = %q\n", ddh.Suit)
	fmt.Printf("\t\t\tCYL_SIZE = %q\n", ddh.CylinderSize)
	fmt.Printf("\t\t\tCYL_WP = %q\n", ddh.CylinderWorkPressure)
	fmt.Printf("\t\t\tCYL_DESC = %q\n", ddh.CylinderDescription)
	fmt.Printf("\t\t\tCYL_START = %q\n", ddh.CylinderStartPressure)
	fmt.Printf("\t\t\tCYL_END = %q\n", ddh.CylinderEndPressure)
	fmt.Printf("\t\t\tCYL_GAS = %q\n", ddh.CylinderGas)
	fmt.Printf("\t\t\tWEIGHT = %q\n", ddh.Weight)
	fmt.Printf("\t\t\tWEIGHT_TYPE = %q\n", ddh.WeightType)
	fmt.Printf("\t\t\tDC_MODEL = %q\n", ddh.DiveComputerModel)
	fmt.Printf("\t\t\tDC_DEVICE_ID = %q\n", ddh.DiveComputerDeviceID)
	fmt.Printf("\t\t\tDC_DIVE_ID = %q\n", ddh.DiveComputerDiveID)
	fmt.Printf("\t\t\tDEPTH_MAX = %q\n", ddh.DepthMax)
	fmt.Printf("\t\t\tDEPTH_MEAN = %q\n", ddh.DepthMean)
	fmt.Printf("\t\t\tTEMP_WATER_MIN = %q\n", ddh.TemperatureWaterMin)
	fmt.Printf("\t\t\tTEMP_AIR = %q\n", ddh.TemperatureAir)
	fmt.Printf("\t\t\tSURFACE_PRESSURE = %q\n", ddh.SurfacePressure)
	return 0
}
