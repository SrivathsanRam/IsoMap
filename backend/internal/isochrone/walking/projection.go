package walking

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const maximumProjectionBytes = 64 << 10

var wktParameterPattern = regexp.MustCompile(`(?i)PARAMETER\s*\[\s*"([^"]+)"\s*,\s*([-+]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[Ee][-+]?[0-9]+)?)\s*\]`)

func readAndValidateProjection(ctx context.Context, file *zip.File) error {
	if file.UncompressedSize64 > maximumProjectionBytes {
		return fmt.Errorf("projection file %q is unexpectedly large (%d bytes)", file.Name, file.UncompressedSize64)
	}
	reader, err := file.Open()
	if err != nil {
		return fmt.Errorf("open projection file %q: %w", file.Name, err)
	}
	defer reader.Close()
	contents, err := readAllContext(ctx, io.LimitReader(reader, maximumProjectionBytes+1))
	if err != nil {
		return fmt.Errorf("read projection file %q: %w", file.Name, err)
	}
	if len(contents) > maximumProjectionBytes {
		return fmt.Errorf("projection file %q exceeds %d bytes", file.Name, maximumProjectionBytes)
	}
	if err := validateSVY21WKT(string(contents)); err != nil {
		return fmt.Errorf("projection file %q: %w", file.Name, err)
	}
	return nil
}

func validateSVY21WKT(wkt string) error {
	normalized := strings.ToLower(strings.Join(strings.Fields(wkt), ""))
	if !strings.Contains(normalized, `projcs["svy21"`) {
		return fmt.Errorf("expected SVY21 projected coordinate system")
	}
	if !strings.Contains(normalized, `projection["transverse_mercator"]`) {
		return fmt.Errorf("expected Transverse_Mercator projection")
	}
	if !strings.Contains(normalized, `geogcs["gcs_wgs_1984"`) ||
		!strings.Contains(normalized, `datum["d_wgs_1984"`) ||
		!strings.Contains(normalized, `spheroid["wgs_1984",6378137`) {
		return fmt.Errorf("expected the WGS 84 geographic datum and spheroid")
	}
	if !strings.Contains(normalized, `unit["meter",1`) {
		return fmt.Errorf("expected metre projection units")
	}

	parameters := make(map[string]float64)
	for _, match := range wktParameterPattern.FindAllStringSubmatch(wkt, -1) {
		value, err := strconv.ParseFloat(match[2], 64)
		if err != nil || !finite(value) {
			return fmt.Errorf("projection parameter %q is not finite", match[1])
		}
		parameters[strings.ToLower(strings.TrimSpace(match[1]))] = value
	}
	expected := []struct {
		name      string
		value     float64
		tolerance float64
	}{
		{name: "central_meridian", value: 103.8333333333333, tolerance: 1e-10},
		{name: "false_easting", value: 28001.642, tolerance: 1e-6},
		{name: "false_northing", value: 38744.572, tolerance: 1e-6},
		{name: "scale_factor", value: 1, tolerance: 1e-12},
		{name: "latitude_of_origin", value: 1.366666666666667, tolerance: 1e-12},
	}
	for _, item := range expected {
		value, ok := parameters[item.name]
		if !ok {
			return fmt.Errorf("missing %s parameter", item.name)
		}
		if math.Abs(value-item.value) > item.tolerance {
			return fmt.Errorf("%s is %.15g; expected %.15g", item.name, value, item.value)
		}
	}
	return nil
}

func readAllContext(ctx context.Context, reader io.Reader) ([]byte, error) {
	result := make([]byte, 0, 4096)
	buffer := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		count, err := reader.Read(buffer)
		if count > 0 {
			result = append(result, buffer[:count]...)
		}
		if err == io.EOF {
			return result, nil
		}
		if err != nil {
			return nil, err
		}
	}
}
