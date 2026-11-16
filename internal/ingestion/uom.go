package ingestion

import (
	"strings"
)

// UOMCategory defines the dimensional domain of a unit.
type UOMCategory string

const (
	CategoryCount  UOMCategory = "COUNT"
	CategoryMass   UOMCategory = "MASS"
	CategoryLength UOMCategory = "LENGTH"
	CategoryVolume UOMCategory = "VOLUME"
)

// UnitConversion contains conversion parameters relative to a base unit.
type UnitConversion struct {
	StandardCode string
	Category     UOMCategory
	BaseFactor   float64 // Multiplication factor to reach base unit
}

// Standard unit registry mapping common procurement aliases to canonical units.
var unitRegistry = map[string]UnitConversion{
	// Count based
	"EA":   {StandardCode: "EA", Category: CategoryCount, BaseFactor: 1.0},
	"EACH": {StandardCode: "EA", Category: CategoryCount, BaseFactor: 1.0},
	"PC":   {StandardCode: "EA", Category: CategoryCount, BaseFactor: 1.0},
	"PCS":  {StandardCode: "EA", Category: CategoryCount, BaseFactor: 1.0},
	"UNIT": {StandardCode: "EA", Category: CategoryCount, BaseFactor: 1.0},
	"UN":   {StandardCode: "EA", Category: CategoryCount, BaseFactor: 1.0},
	"DZ":   {StandardCode: "EA", Category: CategoryCount, BaseFactor: 12.0},
	"DOZ":  {StandardCode: "EA", Category: CategoryCount, BaseFactor: 12.0},
	"GRO":  {StandardCode: "EA", Category: CategoryCount, BaseFactor: 144.0},
	"PACK": {StandardCode: "EA", Category: CategoryCount, BaseFactor: 10.0},
	"PK":   {StandardCode: "EA", Category: CategoryCount, BaseFactor: 10.0},
	"BOX":  {StandardCode: "EA", Category: CategoryCount, BaseFactor: 24.0},
	"BX":   {StandardCode: "EA", Category: CategoryCount, BaseFactor: 24.0},
	"CASE": {StandardCode: "EA", Category: CategoryCount, BaseFactor: 48.0},
	"CS":   {StandardCode: "EA", Category: CategoryCount, BaseFactor: 48.0},

	// Mass based (Base: KG)
	"KG":  {StandardCode: "KG", Category: CategoryMass, BaseFactor: 1.0},
	"KGS": {StandardCode: "KG", Category: CategoryMass, BaseFactor: 1.0},
	"G":   {StandardCode: "KG", Category: CategoryMass, BaseFactor: 0.001},
	"GM":  {StandardCode: "KG", Category: CategoryMass, BaseFactor: 0.001},
	"LB":  {StandardCode: "KG", Category: CategoryMass, BaseFactor: 0.45359237},
	"LBS": {StandardCode: "KG", Category: CategoryMass, BaseFactor: 0.45359237},
	"OZ":  {StandardCode: "KG", Category: CategoryMass, BaseFactor: 0.0283495},
	"TON": {StandardCode: "KG", Category: CategoryMass, BaseFactor: 1000.0},
	"MT":  {StandardCode: "KG", Category: CategoryMass, BaseFactor: 1000.0},

	// Length based (Base: M)
	"M":   {StandardCode: "M", Category: CategoryLength, BaseFactor: 1.0},
	"MTR": {StandardCode: "M", Category: CategoryLength, BaseFactor: 1.0},
	"CM":  {StandardCode: "M", Category: CategoryLength, BaseFactor: 0.01},
	"MM":  {StandardCode: "M", Category: CategoryLength, BaseFactor: 0.001},
	"FT":  {StandardCode: "M", Category: CategoryLength, BaseFactor: 0.3048},
	"IN":  {StandardCode: "M", Category: CategoryLength, BaseFactor: 0.0254},
	"YD":  {StandardCode: "M", Category: CategoryLength, BaseFactor: 0.9144},

	// Volume based (Base: L)
	"L":   {StandardCode: "L", Category: CategoryVolume, BaseFactor: 1.0},
	"LTR": {StandardCode: "L", Category: CategoryVolume, BaseFactor: 1.0},
	"ML":  {StandardCode: "L", Category: CategoryVolume, BaseFactor: 0.001},
	"GAL": {StandardCode: "L", Category: CategoryVolume, BaseFactor: 3.78541},
}

// NormalizeUOM converts a string alias into its standardized uppercase key.
func NormalizeUOM(uom string) string {
	clean := strings.ToUpper(strings.TrimSpace(uom))
	if info, found := unitRegistry[clean]; found {
		return info.StandardCode
	}
	if clean == "" {
		return "EA"
	}
	return clean
}

// ConvertQuantity converts a numeric quantity from source unit to target unit.
// Returns the converted quantity and a boolean indicating if conversion succeeded.
func ConvertQuantity(qty float64, fromUOM, toUOM string) (float64, bool) {
	fromNorm := strings.ToUpper(strings.TrimSpace(fromUOM))
	toNorm := strings.ToUpper(strings.TrimSpace(toUOM))

	if fromNorm == toNorm || (fromNorm == "" && toNorm == "") {
		return qty, true
	}

	fromInfo, fromOk := unitRegistry[fromNorm]
	toInfo, toOk := unitRegistry[toNorm]

	if !fromOk || !toOk {
		return qty, false
	}

	if fromInfo.Category != toInfo.Category {
		return qty, false
	}

	// Normalize to base unit, then convert to target unit
	baseQty := qty * fromInfo.BaseFactor
	targetQty := baseQty / toInfo.BaseFactor
	return targetQty, true
}
