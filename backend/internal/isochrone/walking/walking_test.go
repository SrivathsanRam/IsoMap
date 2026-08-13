package walking

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const fixtureProjection = `PROJCS["SVY21",GEOGCS["GCS_WGS_1984",DATUM["D_WGS_1984",SPHEROID["WGS_1984",6378137.0,298.257223563]],PRIMEM["Greenwich",0.0],UNIT["Degree",0.0174532925199433]],PROJECTION["Transverse_Mercator"],PARAMETER["False_Easting",28001.642],PARAMETER["False_Northing",38744.572],PARAMETER["Central_Meridian",103.8333333333333],PARAMETER["Scale_Factor",1.0],PARAMETER["Latitude_Of_Origin",1.366666666666667],UNIT["Meter",1.0]]`

func TestBuildArchiveRasterizesPolyline(t *testing.T) {
	archivePath := writeFixtureArchive(t, fixtureProjection, fixtureShapefile(t), nil)
	config := DefaultBuildConfig()
	config.CellSizeMeters = 20
	artifact, err := BuildArchive(context.Background(), archivePath, config)
	if err != nil {
		t.Fatalf("BuildArchive() error = %v", err)
	}

	manifest := artifact.Manifest
	if manifest.Format != FormatName || manifest.SchemaVersion != SchemaVersion {
		t.Fatalf("unexpected artifact format: %+v", manifest)
	}
	if manifest.ShapeRecordCount != 2 || manifest.PolylineRecordCount != 1 || manifest.PartCount != 1 || manifest.PointCount != 3 || manifest.SegmentCount != 2 {
		t.Fatalf("unexpected geometry counts: %+v", manifest)
	}
	if manifest.Columns != 2 || manifest.Rows != 2 || manifest.WalkableCellCount != 3 {
		t.Fatalf("unexpected raster counts: %+v", manifest)
	}
	if manifest.ConnectedComponentCount != 1 || manifest.LargestComponentCells != 3 {
		t.Fatalf("unexpected connectivity: %+v", manifest)
	}
	for _, cell := range [][2]int{{0, 0}, {0, 1}, {1, 1}} {
		index, ok := artifact.Network.Index(cell[0], cell[1])
		if !ok || !artifact.Walkable(index) {
			t.Fatalf("cell row=%d col=%d is not walkable", cell[0], cell[1])
		}
	}
	index, _ := artifact.Network.Index(1, 0)
	if artifact.Walkable(index) {
		t.Fatal("untraversed cell is walkable")
	}
	easting, northing, ok := artifact.CellCenter(0, 0)
	if !ok || easting != 28010 || northing != 38750 {
		t.Fatalf("CellCenter(0,0) = %.3f, %.3f, %v", easting, northing, ok)
	}
}

func TestDefaultBuildConfigUsesValidatedResolution(t *testing.T) {
	if got, want := DefaultBuildConfig().CellSizeMeters, 15.0; got != want {
		t.Fatalf("default cell size = %.1f, want %.1f", got, want)
	}
}

func TestArtifactRoundTripAndCorruptionValidation(t *testing.T) {
	archivePath := writeFixtureArchive(t, fixtureProjection, fixtureShapefile(t), nil)
	artifact, err := BuildArchive(context.Background(), archivePath, DefaultBuildConfig())
	if err != nil {
		t.Fatal(err)
	}
	firstPath := filepath.Join(t.TempDir(), "first.walk")
	secondPath := filepath.Join(t.TempDir(), "second.walk")
	if err := WriteArtifact(firstPath, artifact); err != nil {
		t.Fatalf("WriteArtifact(first) error = %v", err)
	}
	if err := WriteArtifact(secondPath, artifact); err != nil {
		t.Fatalf("WriteArtifact(second) error = %v", err)
	}
	first, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("repeated writes are not deterministic")
	}
	loaded, err := LoadArtifact(firstPath)
	if err != nil {
		t.Fatalf("LoadArtifact() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, artifact) {
		t.Fatal("round-tripped artifact differs")
	}

	corrupt := append([]byte(nil), first...)
	corrupt[len(corrupt)-8] ^= 0xff
	corruptPath := filepath.Join(t.TempDir(), "corrupt.walk")
	if err := os.WriteFile(corruptPath, corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadArtifact(corruptPath); err == nil {
		t.Fatal("LoadArtifact() accepted corrupt compressed data")
	}

	inconsistent := *artifact
	inconsistent.Manifest.WalkableCellCount++
	if err := WriteArtifact(filepath.Join(t.TempDir(), "bad.walk"), &inconsistent); err == nil || !strings.Contains(err.Error(), "walkable-cell count") {
		t.Fatalf("WriteArtifact(inconsistent) error = %v", err)
	}
}

func TestBuildArchiveRejectsWrongProjectionAndAmbiguousMembers(t *testing.T) {
	wrongProjection := strings.Replace(fixtureProjection, `PARAMETER["Scale_Factor",1.0]`, `PARAMETER["Scale_Factor",0.9996]`, 1)
	path := writeFixtureArchive(t, wrongProjection, fixtureShapefile(t), nil)
	if _, err := BuildArchive(context.Background(), path, DefaultBuildConfig()); err == nil || !strings.Contains(err.Error(), "scale_factor") {
		t.Fatalf("BuildArchive(wrong projection) error = %v", err)
	}

	path = writeFixtureArchive(t, fixtureProjection, fixtureShapefile(t), map[string][]byte{"duplicate.shp": fixtureShapefile(t)})
	if _, err := BuildArchive(context.Background(), path, DefaultBuildConfig()); err == nil || !strings.Contains(err.Error(), "exactly one .shp") {
		t.Fatalf("BuildArchive(ambiguous archive) error = %v", err)
	}
}

func TestValidateSVY21WKTRejectsRequiredParameterAndDatumChanges(t *testing.T) {
	tests := []struct {
		name        string
		oldValue    string
		newValue    string
		wantMessage string
	}{
		{name: "latitude of origin", oldValue: `PARAMETER["Latitude_Of_Origin",1.366666666666667]`, newValue: `PARAMETER["Latitude_Of_Origin",0]`, wantMessage: "latitude_of_origin"},
		{name: "central meridian", oldValue: `PARAMETER["Central_Meridian",103.8333333333333]`, newValue: `PARAMETER["Central_Meridian",103]`, wantMessage: "central_meridian"},
		{name: "false easting", oldValue: `PARAMETER["False_Easting",28001.642]`, newValue: `PARAMETER["False_Easting",0]`, wantMessage: "false_easting"},
		{name: "false northing", oldValue: `PARAMETER["False_Northing",38744.572]`, newValue: `PARAMETER["False_Northing",0]`, wantMessage: "false_northing"},
		{name: "datum", oldValue: `DATUM["D_WGS_1984"`, newValue: `DATUM["D_NAD_1983"`, wantMessage: "WGS 84"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wkt := strings.Replace(fixtureProjection, test.oldValue, test.newValue, 1)
			if wkt == fixtureProjection {
				t.Fatal("test replacement did not change WKT")
			}
			if err := validateSVY21WKT(wkt); err == nil || !strings.Contains(err.Error(), test.wantMessage) {
				t.Fatalf("validateSVY21WKT() error = %v, want message containing %q", err, test.wantMessage)
			}
		})
	}
}

func TestBuildArchiveRejectsMalformedShapeAndCancellation(t *testing.T) {
	shape := fixtureShapefile(t)
	binary.BigEndian.PutUint32(shape[24:28], uint32(len(shape)/2+1))
	path := writeFixtureArchive(t, fixtureProjection, shape, nil)
	if _, err := BuildArchive(context.Background(), path, DefaultBuildConfig()); err == nil || !strings.Contains(err.Error(), "declares") {
		t.Fatalf("BuildArchive(malformed shape) error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := BuildArchive(ctx, path, DefaultBuildConfig()); err != context.Canceled {
		t.Fatalf("BuildArchive(canceled) error = %v", err)
	}
}

func writeFixtureArchive(t *testing.T, projection string, shape []byte, extras map[string][]byte) string {
	t.Helper()
	var encoded bytes.Buffer
	archive := zip.NewWriter(&encoded)
	writeMember := func(name string, contents []byte) {
		t.Helper()
		writer, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(contents); err != nil {
			t.Fatal(err)
		}
	}
	writeMember("nested/FOOTPATH.SHP", shape)
	writeMember("nested/FOOTPATH.PRJ", []byte(projection))
	for name, contents := range extras {
		writeMember(name, contents)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "footpaths.zip")
	if err := os.WriteFile(path, encoded.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func fixtureShapefile(t *testing.T) []byte {
	t.Helper()
	points := []shapePoint{{x: 28001, y: 38741}, {x: 28021, y: 38741}, {x: 28021, y: 38761}}
	bounds := Bounds{MinEasting: 28001, MinNorthing: 38741, MaxEasting: 28021, MaxNorthing: 38761}
	polyline := encodePolylineRecord(bounds, []int32{0}, points)
	null := make([]byte, 4)
	binary.LittleEndian.PutUint32(null, shapeTypeNull)
	totalBytes := shapeFileHeaderSize + 8 + len(polyline) + 8 + len(null)
	result := make([]byte, shapeFileHeaderSize, totalBytes)
	binary.BigEndian.PutUint32(result[0:4], shapeFileCode)
	binary.BigEndian.PutUint32(result[24:28], uint32(totalBytes/2))
	binary.LittleEndian.PutUint32(result[28:32], shapeFileVersion)
	binary.LittleEndian.PutUint32(result[32:36], shapeTypePolyline)
	putBounds(result[36:68], bounds)
	result = appendShapeRecord(result, 1, polyline)
	result = appendShapeRecord(result, 2, null)
	if len(result) != totalBytes {
		t.Fatalf("fixture length = %d, want %d", len(result), totalBytes)
	}
	return result
}

func encodePolylineRecord(bounds Bounds, parts []int32, points []shapePoint) []byte {
	result := make([]byte, 44+len(parts)*4+len(points)*16)
	binary.LittleEndian.PutUint32(result[0:4], shapeTypePolyline)
	putBounds(result[4:36], bounds)
	binary.LittleEndian.PutUint32(result[36:40], uint32(len(parts)))
	binary.LittleEndian.PutUint32(result[40:44], uint32(len(points)))
	for index, part := range parts {
		binary.LittleEndian.PutUint32(result[44+index*4:48+index*4], uint32(part))
	}
	pointsOffset := 44 + len(parts)*4
	for index, point := range points {
		offset := pointsOffset + index*16
		binary.LittleEndian.PutUint64(result[offset:offset+8], math.Float64bits(point.x))
		binary.LittleEndian.PutUint64(result[offset+8:offset+16], math.Float64bits(point.y))
	}
	return result
}

func appendShapeRecord(destination []byte, number uint32, contents []byte) []byte {
	header := make([]byte, 8)
	binary.BigEndian.PutUint32(header[0:4], number)
	binary.BigEndian.PutUint32(header[4:8], uint32(len(contents)/2))
	destination = append(destination, header...)
	return append(destination, contents...)
}

func putBounds(destination []byte, bounds Bounds) {
	values := []float64{bounds.MinEasting, bounds.MinNorthing, bounds.MaxEasting, bounds.MaxNorthing}
	for index, value := range values {
		binary.LittleEndian.PutUint64(destination[index*8:(index+1)*8], math.Float64bits(value))
	}
}
