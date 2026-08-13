package walking

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

const (
	shapeFileCode       = 9994
	shapeFileVersion    = 1000
	shapeTypeNull       = 0
	shapeTypePolyline   = 3
	shapeFileHeaderSize = 100
)

type shapePoint struct {
	x float64
	y float64
}

type shapeSummary struct {
	bounds          Bounds
	shapeRecords    int
	polylineRecords int
	parts           int
	points          int
	segments        int
	samples         int64
}

func parseAndRasterizeShapefile(ctx context.Context, reader io.Reader, size uint64, config BuildConfig) (Network, shapeSummary, error) {
	var network Network
	var summary shapeSummary
	if size < shapeFileHeaderSize {
		return network, summary, fmt.Errorf("shapefile is %d bytes; header requires %d", size, shapeFileHeaderSize)
	}
	header := make([]byte, shapeFileHeaderSize)
	if err := readFullContext(ctx, reader, header); err != nil {
		return network, summary, fmt.Errorf("read shapefile header: %w", err)
	}
	if code := binary.BigEndian.Uint32(header[0:4]); code != shapeFileCode {
		return network, summary, fmt.Errorf("invalid shapefile code %d", code)
	}
	for offset := 4; offset < 24; offset += 4 {
		if value := binary.BigEndian.Uint32(header[offset : offset+4]); value != 0 {
			return network, summary, fmt.Errorf("reserved shapefile header word at byte %d is non-zero", offset)
		}
	}
	declaredSize := uint64(binary.BigEndian.Uint32(header[24:28])) * 2
	if declaredSize != size {
		return network, summary, fmt.Errorf("shapefile header declares %d bytes but ZIP member has %d", declaredSize, size)
	}
	if version := binary.LittleEndian.Uint32(header[28:32]); version != shapeFileVersion {
		return network, summary, fmt.Errorf("unsupported shapefile version %d", version)
	}
	if shapeType := binary.LittleEndian.Uint32(header[32:36]); shapeType != shapeTypePolyline {
		return network, summary, fmt.Errorf("unsupported shapefile shape type %d; expected PolyLine (3)", shapeType)
	}

	summary.bounds = Bounds{
		MinEasting:  littleFloat64(header[36:44]),
		MinNorthing: littleFloat64(header[44:52]),
		MaxEasting:  littleFloat64(header[52:60]),
		MaxNorthing: littleFloat64(header[60:68]),
	}
	if err := validateBounds(summary.bounds, "shapefile header"); err != nil {
		return network, summary, err
	}
	network, err := newNetwork(summary.bounds, config)
	if err != nil {
		return network, summary, err
	}

	consumed := uint64(shapeFileHeaderSize)
	expectedRecordNumber := uint32(1)
	for consumed < size {
		if err := ctx.Err(); err != nil {
			return Network{}, shapeSummary{}, err
		}
		if size-consumed < 8 {
			return Network{}, shapeSummary{}, fmt.Errorf("truncated shapefile record header at byte %d", consumed)
		}
		recordHeader := make([]byte, 8)
		if err := readFullContext(ctx, reader, recordHeader); err != nil {
			return Network{}, shapeSummary{}, fmt.Errorf("read shapefile record header at byte %d: %w", consumed, err)
		}
		consumed += 8
		recordNumber := binary.BigEndian.Uint32(recordHeader[0:4])
		if recordNumber != expectedRecordNumber {
			return Network{}, shapeSummary{}, fmt.Errorf("shapefile record number %d is out of sequence; expected %d", recordNumber, expectedRecordNumber)
		}
		expectedRecordNumber++
		contentBytes := uint64(binary.BigEndian.Uint32(recordHeader[4:8])) * 2
		if contentBytes < 4 {
			return Network{}, shapeSummary{}, fmt.Errorf("shapefile record %d is shorter than a shape type", recordNumber)
		}
		if contentBytes > size-consumed {
			return Network{}, shapeSummary{}, fmt.Errorf("shapefile record %d declares %d bytes with only %d remaining", recordNumber, contentBytes, size-consumed)
		}
		if contentBytes > uint64(config.MaximumShapefileSize) || contentBytes > uint64(maxInt()) {
			return Network{}, shapeSummary{}, fmt.Errorf("shapefile record %d is too large (%d bytes)", recordNumber, contentBytes)
		}
		contents := make([]byte, int(contentBytes))
		if err := readFullContext(ctx, reader, contents); err != nil {
			return Network{}, shapeSummary{}, fmt.Errorf("read shapefile record %d: %w", recordNumber, err)
		}
		consumed += contentBytes
		summary.shapeRecords++
		if err := parsePolylineRecord(ctx, contents, recordNumber, summary.bounds, &network, &summary); err != nil {
			return Network{}, shapeSummary{}, err
		}
	}
	if consumed != size {
		return Network{}, shapeSummary{}, fmt.Errorf("shapefile parser consumed %d of %d bytes", consumed, size)
	}
	if summary.polylineRecords == 0 || summary.segments == 0 {
		return Network{}, shapeSummary{}, fmt.Errorf("shapefile contains no PolyLine segments")
	}
	return network, summary, nil
}

func parsePolylineRecord(ctx context.Context, contents []byte, recordNumber uint32, fileBounds Bounds, network *Network, summary *shapeSummary) error {
	shapeType := binary.LittleEndian.Uint32(contents[0:4])
	if shapeType == shapeTypeNull {
		if len(contents) != 4 {
			return fmt.Errorf("null shapefile record %d has %d bytes; expected 4", recordNumber, len(contents))
		}
		return nil
	}
	if shapeType != shapeTypePolyline {
		return fmt.Errorf("shapefile record %d has shape type %d; expected PolyLine (3) or Null (0)", recordNumber, shapeType)
	}
	if len(contents) < 44 {
		return fmt.Errorf("PolyLine record %d is only %d bytes; expected at least 44", recordNumber, len(contents))
	}
	recordBounds := Bounds{
		MinEasting:  littleFloat64(contents[4:12]),
		MinNorthing: littleFloat64(contents[12:20]),
		MaxEasting:  littleFloat64(contents[20:28]),
		MaxNorthing: littleFloat64(contents[28:36]),
	}
	if err := validateBounds(recordBounds, fmt.Sprintf("shapefile record %d", recordNumber)); err != nil {
		return err
	}
	if !boundsContain(fileBounds, recordBounds) {
		return fmt.Errorf("shapefile record %d bounds fall outside the file bounds", recordNumber)
	}
	numPartsRaw := int32(binary.LittleEndian.Uint32(contents[36:40]))
	numPointsRaw := int32(binary.LittleEndian.Uint32(contents[40:44]))
	if numPartsRaw <= 0 || numPointsRaw <= 0 {
		return fmt.Errorf("PolyLine record %d has invalid part/point counts %d/%d", recordNumber, numPartsRaw, numPointsRaw)
	}
	numParts, numPoints := int(numPartsRaw), int(numPointsRaw)
	required := uint64(44) + uint64(numParts)*4 + uint64(numPoints)*16
	if required != uint64(len(contents)) {
		return fmt.Errorf("PolyLine record %d has %d bytes; counts require exactly %d", recordNumber, len(contents), required)
	}
	parts := make([]int, numParts)
	partsOffset := 44
	for index := range parts {
		parts[index] = int(int32(binary.LittleEndian.Uint32(contents[partsOffset+index*4 : partsOffset+(index+1)*4])))
		if parts[index] < 0 || parts[index] >= numPoints {
			return fmt.Errorf("PolyLine record %d part %d begins at invalid point %d", recordNumber, index, parts[index])
		}
		if index == 0 && parts[index] != 0 {
			return fmt.Errorf("PolyLine record %d first part begins at point %d, not 0", recordNumber, parts[index])
		}
		if index > 0 && parts[index] <= parts[index-1] {
			return fmt.Errorf("PolyLine record %d part offsets are not strictly increasing", recordNumber)
		}
	}
	points := make([]shapePoint, numPoints)
	pointsOffset := partsOffset + numParts*4
	for index := range points {
		if index&4095 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		offset := pointsOffset + index*16
		points[index] = shapePoint{x: littleFloat64(contents[offset : offset+8]), y: littleFloat64(contents[offset+8 : offset+16])}
		if !finite(points[index].x) || !finite(points[index].y) {
			return fmt.Errorf("PolyLine record %d point %d is not finite", recordNumber, index)
		}
		if !pointInBounds(points[index], recordBounds) || !pointInBounds(points[index], fileBounds) {
			return fmt.Errorf("PolyLine record %d point %d falls outside declared bounds", recordNumber, index)
		}
	}

	for partIndex, start := range parts {
		end := numPoints
		if partIndex+1 < len(parts) {
			end = parts[partIndex+1]
		}
		if end-start < 2 {
			return fmt.Errorf("PolyLine record %d part %d has fewer than two points", recordNumber, partIndex)
		}
		for pointIndex := start; pointIndex < end-1; pointIndex++ {
			samples, err := rasterizeSegment(ctx, network, points[pointIndex], points[pointIndex+1])
			if err != nil {
				return fmt.Errorf("rasterize PolyLine record %d part %d segment %d: %w", recordNumber, partIndex, pointIndex-start, err)
			}
			if summary.samples > math.MaxInt64-samples {
				return fmt.Errorf("rasterized sample count overflows at PolyLine record %d", recordNumber)
			}
			summary.samples += samples
			summary.segments++
		}
	}
	summary.polylineRecords++
	summary.parts += numParts
	summary.points += numPoints
	return nil
}

func newNetwork(bounds Bounds, config BuildConfig) (Network, error) {
	minEasting := math.Floor(bounds.MinEasting/config.CellSizeMeters) * config.CellSizeMeters
	minNorthing := math.Floor(bounds.MinNorthing/config.CellSizeMeters) * config.CellSizeMeters
	columnValue := math.Floor((bounds.MaxEasting-minEasting)/config.CellSizeMeters) + 1
	rowValue := math.Floor((bounds.MaxNorthing-minNorthing)/config.CellSizeMeters) + 1
	if !finite(columnValue) || !finite(rowValue) || columnValue < 1 || rowValue < 1 || columnValue > float64(maxInt()) || rowValue > float64(maxInt()) {
		return Network{}, fmt.Errorf("footpath raster dimensions are invalid")
	}
	columns, rows := int(columnValue), int(rowValue)
	if int64(columns) > math.MaxInt64/int64(rows) {
		return Network{}, fmt.Errorf("footpath raster cell count overflows")
	}
	cellCount := int64(columns) * int64(rows)
	if cellCount > config.MaximumGridCells {
		return Network{}, fmt.Errorf("footpath raster requires %d cells, exceeding configured maximum of %d", cellCount, config.MaximumGridCells)
	}
	if cellCount > int64(maxInt()) {
		return Network{}, fmt.Errorf("footpath raster is too large for this platform")
	}
	return Network{
		MinEasting:     minEasting,
		MinNorthing:    minNorthing,
		CellSizeMeters: config.CellSizeMeters,
		Columns:        columns,
		Rows:           rows,
		WalkableBits:   make([]uint64, (cellCount+63)/64),
	}, nil
}

func readFullContext(ctx context.Context, reader io.Reader, destination []byte) error {
	for len(destination) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		count, err := reader.Read(destination)
		if count > 0 {
			destination = destination[count:]
		}
		if len(destination) == 0 {
			return nil
		}
		if err != nil {
			if err == io.EOF {
				return io.ErrUnexpectedEOF
			}
			return err
		}
		if count == 0 {
			return io.ErrNoProgress
		}
	}
	return nil
}

func littleFloat64(contents []byte) float64 {
	return math.Float64frombits(binary.LittleEndian.Uint64(contents))
}

func validateBounds(bounds Bounds, label string) error {
	if !finite(bounds.MinEasting) || !finite(bounds.MinNorthing) || !finite(bounds.MaxEasting) || !finite(bounds.MaxNorthing) {
		return fmt.Errorf("%s has non-finite bounds", label)
	}
	if bounds.MinEasting > bounds.MaxEasting || bounds.MinNorthing > bounds.MaxNorthing {
		return fmt.Errorf("%s has inverted bounds", label)
	}
	return nil
}

func boundsContain(outer, inner Bounds) bool {
	const tolerance = 1e-6
	return inner.MinEasting >= outer.MinEasting-tolerance && inner.MinNorthing >= outer.MinNorthing-tolerance &&
		inner.MaxEasting <= outer.MaxEasting+tolerance && inner.MaxNorthing <= outer.MaxNorthing+tolerance
}

func pointInBounds(point shapePoint, bounds Bounds) bool {
	const tolerance = 1e-6
	return point.x >= bounds.MinEasting-tolerance && point.x <= bounds.MaxEasting+tolerance &&
		point.y >= bounds.MinNorthing-tolerance && point.y <= bounds.MaxNorthing+tolerance
}

func maxInt() int {
	return int(^uint(0) >> 1)
}
