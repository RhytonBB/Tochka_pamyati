package service

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

const unknownRegion = "Неизвестный регион"

type geoPoint struct {
	Lon float64
	Lat float64
}

type geoRing []geoPoint

type geoPolygon []geoRing

type regionGeometry struct {
	Name     string
	Code     string
	MinLon   float64
	MinLat   float64
	MaxLon   float64
	MaxLat   float64
	Polygons []geoPolygon
}

type geoJSONCollection struct {
	Features []geoJSONFeature `json:"features"`
}

type geoJSONFeature struct {
	Properties geoJSONProperties `json:"properties"`
	Geometry   geoJSONGeometry   `json:"geometry"`
}

type geoJSONProperties struct {
	ID         any    `json:"id"`
	LocName    string `json:"locname"`
	Name       string `json:"name"`
	AdminLevel int    `json:"adminlevel"`
}

type geoJSONGeometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

type GeographyService struct {
	regions []regionGeometry
	names   []string
}

func NewGeographyService(geoJSONPath string) (*GeographyService, error) {
	if strings.TrimSpace(geoJSONPath) == "" {
		return nil, errors.New("regions geojson path is required")
	}

	payload, err := os.ReadFile(geoJSONPath)
	if err != nil {
		return nil, err
	}

	var collection geoJSONCollection
	if err := json.Unmarshal(payload, &collection); err != nil {
		return nil, err
	}

	regions := make([]regionGeometry, 0, len(collection.Features))
	uniqueNames := map[string]struct{}{}
	nameFieldCounts := map[string]int{
		"name":    0,
		"locname": 0,
	}

	for _, feature := range collection.Features {
		if feature.Properties.AdminLevel != 4 {
			continue
		}

		name, fieldUsed := pickRegionName(feature.Properties)
		if name == "" {
			continue
		}

		polygons, bbox, err := decodeRegionGeometry(feature.Geometry)
		if err != nil || len(polygons) == 0 {
			continue
		}

		code := normalizeRegionCode(feature.Properties.ID)
		regions = append(regions, regionGeometry{
			Name:     name,
			Code:     code,
			MinLon:   bbox[0],
			MinLat:   bbox[1],
			MaxLon:   bbox[2],
			MaxLat:   bbox[3],
			Polygons: polygons,
		})
		uniqueNames[name] = struct{}{}
		nameFieldCounts[fieldUsed]++
	}

	names := make([]string, 0, len(uniqueNames))
	for name := range uniqueNames {
		names = append(names, name)
	}
	sort.Strings(names)

	log.Printf(
		"[REGIONS] Загружено регионов: %d, поле name: %d, поле locname: %d, примеры: %s",
		len(regions),
		nameFieldCounts["name"],
		nameFieldCounts["locname"],
		strings.Join(sampleNames(names, 5), ", "),
	)

	return &GeographyService{
		regions: regions,
		names:   names,
	}, nil
}

func (s *GeographyService) GetRegionByCoords(lon, lat float64) string {
	for _, region := range s.regions {
		if lon < region.MinLon || lon > region.MaxLon || lat < region.MinLat || lat > region.MaxLat {
			continue
		}
		if pointInRegion(lon, lat, region) {
			return region.Name
		}
	}
	return unknownRegion
}

func (s *GeographyService) GetAllRegions() []string {
	out := make([]string, len(s.names))
	copy(out, s.names)
	return out
}

func decodeRegionGeometry(geometry geoJSONGeometry) ([]geoPolygon, [4]float64, error) {
	switch geometry.Type {
	case "Polygon":
		var coords [][][]float64
		if err := json.Unmarshal(geometry.Coordinates, &coords); err != nil {
			return nil, [4]float64{}, err
		}
		polygon, bbox := convertPolygon(coords)
		return []geoPolygon{polygon}, bbox, nil
	case "MultiPolygon":
		var coords [][][][]float64
		if err := json.Unmarshal(geometry.Coordinates, &coords); err != nil {
			return nil, [4]float64{}, err
		}
		polygons := make([]geoPolygon, 0, len(coords))
		bbox := [4]float64{180, 90, -180, -90}
		for _, polygonCoords := range coords {
			polygon, polyBBox := convertPolygon(polygonCoords)
			if len(polygon) == 0 {
				continue
			}
			polygons = append(polygons, polygon)
			mergeBBox(&bbox, polyBBox)
		}
		return polygons, bbox, nil
	default:
		return nil, [4]float64{}, errors.New("unsupported geometry type")
	}
}

func convertPolygon(coords [][][]float64) (geoPolygon, [4]float64) {
	polygon := make(geoPolygon, 0, len(coords))
	bbox := [4]float64{180, 90, -180, -90}

	for _, ringCoords := range coords {
		ring := make(geoRing, 0, len(ringCoords))
		for _, rawPoint := range ringCoords {
			if len(rawPoint) < 2 {
				continue
			}
			point := geoPoint{Lon: rawPoint[0], Lat: rawPoint[1]}
			ring = append(ring, point)
			updateBBox(&bbox, point)
		}
		if len(ring) >= 4 {
			polygon = append(polygon, ring)
		}
	}

	return polygon, bbox
}

func pointInRegion(lon, lat float64, region regionGeometry) bool {
	point := geoPoint{Lon: lon, Lat: lat}
	for _, polygon := range region.Polygons {
		if len(polygon) == 0 {
			continue
		}
		if !pointInRing(point, polygon[0]) {
			continue
		}
		inHole := false
		for i := 1; i < len(polygon); i++ {
			if pointInRing(point, polygon[i]) {
				inHole = true
				break
			}
		}
		if !inHole {
			return true
		}
	}
	return false
}

func pointInRing(point geoPoint, ring geoRing) bool {
	inside := false
	j := len(ring) - 1
	for i := 0; i < len(ring); i++ {
		pi := ring[i]
		pj := ring[j]
		intersects := ((pi.Lat > point.Lat) != (pj.Lat > point.Lat)) &&
			(point.Lon < (pj.Lon-pi.Lon)*(point.Lat-pi.Lat)/(pj.Lat-pi.Lat)+pi.Lon)
		if intersects {
			inside = !inside
		}
		j = i
	}
	return inside
}

func updateBBox(bbox *[4]float64, point geoPoint) {
	if point.Lon < bbox[0] {
		bbox[0] = point.Lon
	}
	if point.Lat < bbox[1] {
		bbox[1] = point.Lat
	}
	if point.Lon > bbox[2] {
		bbox[2] = point.Lon
	}
	if point.Lat > bbox[3] {
		bbox[3] = point.Lat
	}
}

func mergeBBox(target *[4]float64, source [4]float64) {
	if source[0] < target[0] {
		target[0] = source[0]
	}
	if source[1] < target[1] {
		target[1] = source[1]
	}
	if source[2] > target[2] {
		target[2] = source[2]
	}
	if source[3] > target[3] {
		target[3] = source[3]
	}
}

func normalizeRegionName(value string) string {
	return strings.TrimSpace(value)
}

func normalizeRegionCode(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}

func pickRegionName(props geoJSONProperties) (string, string) {
	if name := normalizeRegionName(props.Name); name != "" {
		return name, "name"
	}
	if name := normalizeRegionName(props.LocName); name != "" {
		return name, "locname"
	}
	return "", ""
}

func sampleNames(items []string, limit int) []string {
	if len(items) <= limit {
		return items
	}
	out := make([]string, limit)
	copy(out, items[:limit])
	return out
}
