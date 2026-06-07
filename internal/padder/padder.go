package padder

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"imagesplitter/internal/config"
	"imagesplitter/internal/logging"
	"imagesplitter/internal/models"
)

// Side indicates which side the white space is added to.
type Side string

const (
	SideLeft  Side = "left"  // white space on left,  image on right
	SideRight Side = "right" // white space on right, image on left
)

// ProcessFolder processes all padding targets in dir.
// Appends padding ImageResults to an existing FolderResult so both
// splitting and padding results live together under the same folder.
func ProcessFolder(result *models.FolderResult, cfg *config.Config, logger *logging.Logger) {
	defer func() {
		if r := recover(); r != nil {
			result.ImageResults = append(result.ImageResults, &models.ImageResult{
				FileName:  "unknown",
				Operation: models.OperationPadding,
				Status:    models.StatusError,
				Message:   fmt.Sprintf("unexpected panic: %v", r),
			})
		}
	}()

	dir := result.FolderPath

	for _, baseName := range cfg.Padding.LeftPadNames {
		logger.Debug(fmt.Sprintf("[Pad] Folder=%q LEFT padding for %q", dir, baseName))
		results := processBase(dir, baseName, SideLeft, cfg, logger)
		result.ImageResults = append(result.ImageResults, results...)
	}

	for _, baseName := range cfg.Padding.RightPadNames {
		logger.Debug(fmt.Sprintf("[Pad] Folder=%q RIGHT padding for %q", dir, baseName))
		results := processBase(dir, baseName, SideRight, cfg, logger)
		result.ImageResults = append(result.ImageResults, results...)
	}

	result.EndTime = time.Now()
}

func processBase(dir, baseName string, side Side, cfg *config.Config, logger *logging.Logger) []*models.ImageResult {
	var results []*models.ImageResult
	found := false

	for _, ext := range config.SupportedExtensions {
		candidate := filepath.Join(dir, baseName+ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			continue
		}
		found = true
		logger.Debug(fmt.Sprintf("[Pad] Found %q side=%s", candidate, side))
		ir := processSingleImage(candidate, ext, side, cfg, logger)
		results = append(results, ir)
	}

	if !found {
		sideLabel := "left"
		if side == SideRight {
			sideLabel = "right"
		}
		results = append(results, &models.ImageResult{
			FileName:  baseName + ".*",
			Operation: models.OperationPadding,
			Status:    models.StatusTargetImageMissing,
			Message:   fmt.Sprintf("No supported image found for %q (pad %s)", baseName, sideLabel),
		})
	}

	return results
}

func processSingleImage(sourcePath, ext string, side Side, cfg *config.Config, logger *logging.Logger) *models.ImageResult {
	fileName := filepath.Base(sourcePath)
	ir := &models.ImageResult{
		FileName:  fileName,
		Operation: models.OperationPadding,
	}

	// Determine output path.
	outputPath := sourcePath // default: overwrite
	if cfg.Padding.CreateNewFile {
		suffix := cfg.Padding.LeftSuffix
		if side == SideRight {
			suffix = cfg.Padding.RightSuffix
		}
		base := strings.TrimSuffix(sourcePath, ext)
		outputPath = base + suffix + ext

		// Skip if output already exists and overwriteExisting=false.
		if !cfg.Padding.OverwriteExisting {
			if _, err := os.Stat(outputPath); err == nil {
				ir.Status = models.StatusAlreadyProcessed
				ir.Message = fmt.Sprintf("Output already exists: %s", filepath.Base(outputPath))
				return ir
			}
		}
	}

	// Load source image — preserving original format.
	img, err := loadImage(sourcePath, ext)
	if err != nil {
		ir.Status = models.StatusError
		ir.Message = fmt.Sprintf("Could not open image: %v", err)
		return ir
	}

	b := img.Bounds()
	origW := b.Max.X - b.Min.X
	origH := b.Max.Y - b.Min.Y

	logger.Debug(fmt.Sprintf("[Pad] %q original=%dx%d side=%s", fileName, origW, origH, side))

	// Parse pad colour.
	padCol, err := parseHexColor(cfg.Padding.PadColor)
	if err != nil {
		padCol = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		logger.Warn(fmt.Sprintf("[Pad] Invalid padColor %q, defaulting to white", cfg.Padding.PadColor))
	}

	// Create canvas: double width, same height.
	totalW := origW * 2
	canvas := image.NewRGBA(image.Rect(0, 0, totalW, origH))

	// Fill entire canvas with pad colour.
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{padCol}, image.Point{}, draw.Src)

	// Place original image on the correct side.
	var destPt image.Point
	if side == SideLeft {
		// White space on left → image on right half.
		destPt = image.Point{origW, 0}
	} else {
		// White space on right → image on left half.
		destPt = image.Point{0, 0}
	}
	draw.Draw(canvas, image.Rect(destPt.X, destPt.Y, destPt.X+origW, destPt.Y+origH),
		img, b.Min, draw.Src)

	// Save — same format as source.
	if err := saveImage(canvas, outputPath, ext); err != nil {
		ir.Status = models.StatusError
		ir.Message = fmt.Sprintf("Could not save padded image: %v", err)
		return ir
	}

	sideLabel := "left"
	if side == SideRight {
		sideLabel = "right"
	}

	ir.Status = models.StatusProcessed
	if cfg.Padding.CreateNewFile {
		ir.Message = fmt.Sprintf("White space added to %s → saved as %s (%dx%d)",
			sideLabel, filepath.Base(outputPath), totalW, origH)
	} else {
		ir.Message = fmt.Sprintf("White space added to %s, original overwritten (%dx%d)",
			sideLabel, totalW, origH)
	}
	return ir
}

// ── Image I/O — format in = format out, zero conversion ──────────────────────

func loadImage(path, ext string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return jpeg.Decode(f)
	case ".png":
		return png.Decode(f)
	default:
		img, _, err := image.Decode(f)
		return img, err
	}
}

func saveImage(img image.Image, path, ext string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		// JPEG stays JPEG — same quality setting as the splitter.
		return jpeg.Encode(f, img, &jpeg.Options{Quality: 95})
	case ".png":
		return png.Encode(f, img)
	default:
		// BMP, TIFF, WebP: re-encode via PNG (lossless, no quality loss).
		return png.Encode(f, img)
	}
}

// parseHexColor parses a "#RRGGBB" hex string into a color.RGBA.
func parseHexColor(s string) (color.RGBA, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return color.RGBA{}, fmt.Errorf("invalid hex colour: %q", s)
	}
	var r, g, b uint8
	_, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return color.RGBA{}, err
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}, nil
}
