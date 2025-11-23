package server

const (
	UnlabeledRegion            = "Unlabeled Region"
	PrefixForTagsInDescription = "tags:"
	RegionTagPrefix            = "_region_"
)

var CylinderTypeMappings = map[string]string{
	"AL100": "aluminium",
	"HP100": "steel",
	"HP130": "steel",
}

var SpecialTagValueMappings = map[string]string{
	"europe":        "Europe",
	"asia":          "Asia",
	"north-america": "North America",
	"atlantic":      "Atlantic Ocean",
	"indian":        "Indian Ocean",
	"pacific":       "Pacific Ocean",
	"mediterranean": "Mediterranean Sea",
	"red-sea":       "Red Sea",
}

var AwardMappings = map[string]string{
	"1st-dive":            "First dive!",
	"1st-seawater-dive":   "First seawater dive!",
	"1st-shark-encounter": "First shark encounter!",
	"1st-night-dive":      "First night dive!",
	"1st-nitrox-dive":     "First nitrox dive!",
	"100th-dive":          "100th dive!",
}
