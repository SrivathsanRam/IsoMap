package contour

import (
	"fmt"
	"math"
	"sort"

	"github.com/SrivathsanRam/IsoMap/internal/isochrone/spatial"
)

const contourEqualityEpsilon = 1e-7

// MultiPolygon is an RFC 7946 GeoJSON geometry. Positions are [longitude,
// latitude], polygons contain an exterior ring followed by zero or more holes.
type MultiPolygon struct {
	Type        string          `json:"type"`
	Coordinates [][][][]float64 `json:"coordinates"`
}

type Config struct {
	MinimumRingAreaSquareMeters float64
}

func DefaultConfig(cellSizeMeters float64) Config {
	return Config{MinimumRingAreaSquareMeters: cellSizeMeters * cellSizeMeters}
}

// MarchingSquares extracts the grid threshold, stitches segments into rings,
// classifies nested rings as holes, and returns a GeoJSON MultiPolygon.
func MarchingSquares(grid *spatial.Grid, config Config) (MultiPolygon, error) {
	if grid == nil {
		return MultiPolygon{}, fmt.Errorf("travel-time grid is nil")
	}
	if grid.Columns < 2 || grid.Rows < 2 || len(grid.Values) != grid.Columns*grid.Rows {
		return MultiPolygon{}, fmt.Errorf("travel-time grid dimensions are invalid")
	}
	if grid.CellSizeMeters <= 0 || math.IsNaN(grid.CellSizeMeters) || math.IsInf(grid.CellSizeMeters, 0) {
		return MultiPolygon{}, fmt.Errorf("travel-time grid cell size is invalid")
	}
	if config.MinimumRingAreaSquareMeters < 0 || math.IsNaN(config.MinimumRingAreaSquareMeters) || math.IsInf(config.MinimumRingAreaSquareMeters, 0) {
		return MultiPolygon{}, fmt.Errorf("minimum ring area must be finite and non-negative")
	}

	segments := extractSegments(grid)
	if len(segments) == 0 {
		return MultiPolygon{Type: "MultiPolygon", Coordinates: [][][][]float64{}}, nil
	}
	rings, err := stitchRings(segments)
	if err != nil {
		return MultiPolygon{}, err
	}
	rings = filterRings(rings, config.MinimumRingAreaSquareMeters)
	return assembleMultiPolygon(grid, rings), nil
}

type localVertex struct {
	key   pointKey
	point spatial.LocalPoint
}

type pointKey struct {
	x int64
	y int64
}

type segment struct {
	a localVertex
	b localVertex
}

type ring struct {
	points     []spatial.LocalPoint
	signedArea float64
	area       float64
	parent     int
	depth      int
}

func extractSegments(grid *spatial.Grid) []segment {
	segments := make([]segment, 0)
	level := grid.ThresholdSeconds
	for row := 0; row < grid.Rows-1; row++ {
		for column := 0; column < grid.Columns-1; column++ {
			values := [4]float64{
				adjustEquality(grid.Value(column, row), level),
				adjustEquality(grid.Value(column+1, row), level),
				adjustEquality(grid.Value(column+1, row+1), level),
				adjustEquality(grid.Value(column, row+1), level),
			}
			caseIndex := 0
			for index, value := range values {
				if value < level {
					caseIndex |= 1 << index
				}
			}
			if caseIndex == 0 || caseIndex == 15 {
				continue
			}

			points := [4]spatial.LocalPoint{
				grid.LocalPoint(column, row),
				grid.LocalPoint(column+1, row),
				grid.LocalPoint(column+1, row+1),
				grid.LocalPoint(column, row+1),
			}
			edges := [4]localVertex{
				interpolate(points[0], points[1], values[0], values[1], level),
				interpolate(points[1], points[2], values[1], values[2], level),
				interpolate(points[3], points[2], values[3], values[2], level),
				interpolate(points[0], points[3], values[0], values[3], level),
			}
			centerInside := (values[0]+values[1]+values[2]+values[3])/4 < level
			for _, pair := range segmentPairs(caseIndex, centerInside) {
				segments = append(segments, segment{a: edges[pair[0]], b: edges[pair[1]]})
			}
		}
	}
	return segments
}

func adjustEquality(value, level float64) float64 {
	if math.Abs(value-level) < contourEqualityEpsilon {
		return level + contourEqualityEpsilon
	}
	return value
}

func interpolate(a, b spatial.LocalPoint, aValue, bValue, level float64) localVertex {
	denominator := bValue - aValue
	fraction := 0.5
	if denominator != 0 {
		fraction = (level - aValue) / denominator
	}
	fraction = math.Max(0, math.Min(1, fraction))
	point := spatial.LocalPoint{
		X: a.X + fraction*(b.X-a.X),
		Y: a.Y + fraction*(b.Y-a.Y),
	}
	return localVertex{key: quantize(point), point: point}
}

func quantize(point spatial.LocalPoint) pointKey {
	// Micrometre quantization joins the two independently evaluated sides of a
	// shared grid edge without affecting map-scale geometry.
	return pointKey{
		x: int64(math.Round(point.X * 1e6)),
		y: int64(math.Round(point.Y * 1e6)),
	}
}

func segmentPairs(caseIndex int, centerInside bool) [][2]int {
	switch caseIndex {
	case 1:
		return [][2]int{{3, 0}}
	case 2:
		return [][2]int{{0, 1}}
	case 3:
		return [][2]int{{3, 1}}
	case 4:
		return [][2]int{{1, 2}}
	case 5:
		if centerInside {
			return [][2]int{{0, 1}, {2, 3}}
		}
		return [][2]int{{3, 0}, {1, 2}}
	case 6:
		return [][2]int{{0, 2}}
	case 7:
		return [][2]int{{3, 2}}
	case 8:
		return [][2]int{{2, 3}}
	case 9:
		return [][2]int{{2, 0}}
	case 10:
		if centerInside {
			return [][2]int{{3, 0}, {1, 2}}
		}
		return [][2]int{{0, 1}, {2, 3}}
	case 11:
		return [][2]int{{1, 2}}
	case 12:
		return [][2]int{{1, 3}}
	case 13:
		return [][2]int{{0, 1}}
	case 14:
		return [][2]int{{3, 0}}
	default:
		return nil
	}
}

func stitchRings(segments []segment) ([]ring, error) {
	adjacency := make(map[pointKey][]int, len(segments))
	vertices := make(map[pointKey]spatial.LocalPoint, len(segments))
	for index, item := range segments {
		if item.a.key == item.b.key {
			continue
		}
		adjacency[item.a.key] = append(adjacency[item.a.key], index)
		adjacency[item.b.key] = append(adjacency[item.b.key], index)
		vertices[item.a.key] = item.a.point
		vertices[item.b.key] = item.b.point
	}
	for key, connected := range adjacency {
		if len(connected) != 2 {
			return nil, fmt.Errorf("marching-squares contour is not manifold at (%d, %d): degree %d", key.x, key.y, len(connected))
		}
	}

	visited := make([]bool, len(segments))
	rings := make([]ring, 0)
	for startSegment := range segments {
		if visited[startSegment] || segments[startSegment].a.key == segments[startSegment].b.key {
			continue
		}
		startKey := segments[startSegment].a.key
		currentKey := segments[startSegment].b.key
		previousSegment := startSegment
		visited[startSegment] = true
		points := []spatial.LocalPoint{segments[startSegment].a.point, segments[startSegment].b.point}

		for currentKey != startKey {
			connected := adjacency[currentKey]
			nextSegment := connected[0]
			if nextSegment == previousSegment {
				nextSegment = connected[1]
			}
			if visited[nextSegment] {
				return nil, fmt.Errorf("marching-squares contour closes into an already visited segment")
			}
			visited[nextSegment] = true
			next := segments[nextSegment]
			if next.a.key == currentKey {
				currentKey = next.b.key
			} else {
				currentKey = next.a.key
			}
			points = append(points, vertices[currentKey])
			previousSegment = nextSegment
			if len(points) > len(segments)+1 {
				return nil, fmt.Errorf("marching-squares contour did not close")
			}
		}
		if len(points) >= 4 {
			signed := signedArea(points)
			rings = append(rings, ring{
				points:     points,
				signedArea: signed,
				area:       math.Abs(signed),
				parent:     -1,
			})
		}
	}
	return rings, nil
}

func filterRings(rings []ring, minimumArea float64) []ring {
	filtered := rings[:0]
	for _, candidate := range rings {
		if candidate.area >= minimumArea {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func assembleMultiPolygon(grid *spatial.Grid, rings []ring) MultiPolygon {
	if len(rings) == 0 {
		return MultiPolygon{Type: "MultiPolygon", Coordinates: [][][][]float64{}}
	}

	order := make([]int, len(rings))
	for index := range order {
		order[index] = index
	}
	sort.SliceStable(order, func(i, j int) bool {
		return rings[order[i]].area > rings[order[j]].area
	})
	for orderIndex := 1; orderIndex < len(order); orderIndex++ {
		childIndex := order[orderIndex]
		childPoint := rings[childIndex].points[0]
		parentArea := math.Inf(1)
		for candidateOrder := 0; candidateOrder < orderIndex; candidateOrder++ {
			candidateIndex := order[candidateOrder]
			if rings[candidateIndex].area < parentArea && pointInRing(childPoint, rings[candidateIndex].points) {
				rings[childIndex].parent = candidateIndex
				parentArea = rings[candidateIndex].area
			}
		}
		if rings[childIndex].parent >= 0 {
			rings[childIndex].depth = rings[rings[childIndex].parent].depth + 1
		}
	}

	outerOrder := make([]int, 0)
	for _, index := range order {
		if rings[index].depth%2 == 0 {
			outerOrder = append(outerOrder, index)
		}
	}
	coordinates := make([][][][]float64, 0, len(outerOrder))
	for _, outerIndex := range outerOrder {
		polygon := [][][]float64{geoJSONRing(grid, rings[outerIndex], true)}
		for _, candidateIndex := range order {
			candidate := rings[candidateIndex]
			if candidate.parent == outerIndex && candidate.depth%2 == 1 {
				polygon = append(polygon, geoJSONRing(grid, candidate, false))
			}
		}
		coordinates = append(coordinates, polygon)
	}
	return MultiPolygon{Type: "MultiPolygon", Coordinates: coordinates}
}

func geoJSONRing(grid *spatial.Grid, candidate ring, exterior bool) [][]float64 {
	points := candidate.points
	shouldReverse := (exterior && candidate.signedArea < 0) || (!exterior && candidate.signedArea > 0)
	coordinates := make([][]float64, 0, len(points))
	if shouldReverse {
		for index := len(points) - 1; index >= 0; index-- {
			point := grid.GeographicPoint(points[index])
			coordinates = append(coordinates, []float64{point.Lon, point.Lat})
		}
		return coordinates
	}
	for _, local := range points {
		point := grid.GeographicPoint(local)
		coordinates = append(coordinates, []float64{point.Lon, point.Lat})
	}
	return coordinates
}

func signedArea(points []spatial.LocalPoint) float64 {
	area := 0.0
	for index := 0; index < len(points)-1; index++ {
		area += points[index].X*points[index+1].Y - points[index+1].X*points[index].Y
	}
	return area / 2
}

func pointInRing(point spatial.LocalPoint, points []spatial.LocalPoint) bool {
	inside := false
	for current, previous := 0, len(points)-2; current < len(points)-1; previous, current = current, current+1 {
		a := points[current]
		b := points[previous]
		crosses := (a.Y > point.Y) != (b.Y > point.Y)
		if crosses && point.X < (b.X-a.X)*(point.Y-a.Y)/(b.Y-a.Y)+a.X {
			inside = !inside
		}
	}
	return inside
}
