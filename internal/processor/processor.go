package processor

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
	"imagesplitter/internal/filesystem"
	"imagesplitter/internal/logging"
	"imagesplitter/internal/models"
)

// ProcessFolder scans dir for every splitting target base name × every supported extension.
// Always returns a FolderResult — never panics or propagates errors upward.
func ProcessFolder(dir string, cfg *config.Config, logger *logging.Logger) *models.FolderResult {
	result := &models.FolderResult{
		FolderName: filepath.Base(dir),
		FolderPath: dir,
		StartTime:  time.Now(),
	}

	defer func() {
		result.EndTime = time.Now()
		if r := recover(); r != nil {
			result.ImageResults = append(result.ImageResults, &models.ImageResult{
				FileName:  "unknown",
				Operation: models.OperationSplitting,
				Status:    models.StatusError,
				Message:   fmt.Sprintf("unexpected panic: %v", r),
			})
		}
	}()

	for _, baseName := range cfg.Splitting.TargetBaseNames {
		logger.Debug(fmt.Sprintf("[Split] Folder=%q looking for %q", dir, baseName))
		results := processBaseName(dir, baseName, cfg, logger)
		result.ImageResults = append(result.ImageResults, results...)
	}

	return result
}

func processBaseName(dir, baseName string, cfg *config.Config, logger *logging.Logger) []*models.ImageResult {
	var results []*models.ImageResult
	found := false

	for _, ext := range config.SupportedExtensions {
		candidate := filepath.Join(dir, baseName+ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			continue
		}
		found = true
		logger.Debug(fmt.Sprintf("[Split] Found %q", candidate))
		ir := processSingleImage(candidate, ext, cfg, logger)
		results = append(results, ir)
	}

	if !found {
		results = append(results, &models.ImageResult{
			FileName:  baseName + ".*",
			Operation: models.OperationSplitting,
			Status:    models.StatusTargetImageMissing,
			Message:   fmt.Sprintf("No supported image found for %q", baseName),
		})
	}

	return results
}

func processSingleImage(sourcePath, ext string, cfg *config.Config, logger *logging.Logger) *models.ImageResult {
	fileName := filepath.Base(sourcePath)
	ir := &models.ImageResult{
		FileName:  fileName,
		Operation: models.OperationSplitting,
	}

	leftPath, rightPath := filesystem.OutputPaths(sourcePath, cfg.Splitting.LeftSuffix, cfg.Splitting.RightSuffix)

	if !cfg.Splitting.OverwriteExisting && filesystem.ExistsAny(leftPath, rightPath) {
		ir.Status = models.StatusAlreadyProcessed
		ir.Message = "Output files already exist"
		return ir
	}

	img, err := loadImage(sourcePath, ext)
	if err != nil {
		ir.Status = models.StatusError
		ir.Message = fmt.Sprintf("Could not open image: %v", err)
		return ir
	}

	b := img.Bounds()
	w := b.Max.X - b.Min.X
	h := b.Max.Y - b.Min.Y
	logger.Debug(fmt.Sprintf("[Split] %q dimensions=%dx%d splitAt=%d", fileName, w, h, w/2))

	left, right := splitVertically(img)

	if err := saveImage(left, leftPath, ext); err != nil {
		ir.Status = models.StatusError
		ir.Message = fmt.Sprintf("Could not save left image: %v", err)
		return ir
	}
	if err := saveImage(right, rightPath, ext); err != nil {
		ir.Status = models.StatusError
		ir.Message = fmt.Sprintf("Could not save right image: %v", err)
		return ir
	}

	if cfg.Splitting.DeleteOriginal {
		if err := os.Remove(sourcePath); err != nil {
			ir.Status = models.StatusProcessed
			ir.Message = fmt.Sprintf("2 files created; warning: could not delete original: %v", err)
			return ir
		}
	}

	ir.Status = models.StatusProcessed
	ir.Message = fmt.Sprintf("Split into 2 files (%dx%d → %dx%d + %dx%d)", w, h, w/2, h, w-w/2, h)
	return ir
}

// splitVertically splits img into left and right halves.
func splitVertically(img image.Image) (left, right image.Image) {
	b := img.Bounds()
	w := b.Max.X - b.Min.X
	h := b.Max.Y - b.Min.Y
	wL := w / 2
	wR := w - wL

	leftImg := image.NewRGBA(image.Rect(0, 0, wL, h))
	rightImg := image.NewRGBA(image.Rect(0, 0, wR, h))

	draw.Draw(leftImg, leftImg.Bounds(), img, image.Point{b.Min.X, b.Min.Y}, draw.Src)
	draw.Draw(rightImg, rightImg.Bounds(), img, image.Point{b.Min.X + wL, b.Min.Y}, draw.Src)

	return leftImg, rightImg
}

// ── Image I/O ─────────────────────────────────────────────────────────────────

func LoadImage(path string) (image.Image, string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	f, err := os.Open(path)
	if err != nil {
		return nil, ext, err
	}
	defer f.Close()

	switch ext {
	case ".jpg", ".jpeg":
		img, err := jpeg.Decode(f)
		return img, ext, err
	case ".png":
		img, err := png.Decode(f)
		return img, ext, err
	default:
		// BMP, TIFF, WebP: use registered decoders (pulled in via Fyne deps).
		img, _, err := image.Decode(f)
		return img, ext, err
	}
}

func loadImage(path, ext string) (image.Image, error) {
	img, _, err := LoadImage(path)
	return img, err
}

// SaveImage writes an image preserving its original format exactly.
// JPEG→JPEG, JPG→JPG, PNG→PNG — zero conversion.
func SaveImage(img image.Image, path, ext string) error {
	return saveImage(img, path, ext)
}

func saveImage(img image.Image, path, ext string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return jpeg.Encode(f, img, &jpeg.Options{Quality: 95})
	case ".png":
		return png.Encode(f, img)
	default:
		// BMP, TIFF, WebP: encode as PNG to preserve lossless quality.
		// The file extension is kept as-is so the filename is unchanged.
		return png.Encode(f, img)
	}
}

// ── BMP helpers (pure stdlib, no extra dependency) ────────────────────────────

func EncodeBMP(f *os.File, img image.Image) error {
	b := img.Bounds()
	w := b.Max.X - b.Min.X
	h := b.Max.Y - b.Min.Y

	rowSize := ((w*3 + 3) / 4) * 4
	pixelDataSize := rowSize * h
	fileSize := 54 + pixelDataSize

	header := make([]byte, 54)
	header[0] = 'B'; header[1] = 'M'
	putU32LE(header[2:], uint32(fileSize))
	putU32LE(header[10:], 54)
	putU32LE(header[14:], 40)
	putU32LE(header[18:], uint32(w))
	putU32LE(header[22:], uint32(h))
	header[26] = 1; header[28] = 24
	putU32LE(header[34:], uint32(pixelDataSize))

	if _, err := f.Write(header); err != nil {
		return err
	}

	row := make([]byte, rowSize)
	for y := h - 1; y >= 0; y-- {
		for x := 0; x < w; x++ {
			r, g, bl, _ := color.RGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).RGBA()
			row[x*3+0] = byte(bl >> 8)
			row[x*3+1] = byte(g >> 8)
			row[x*3+2] = byte(r >> 8)
		}
		for i := w * 3; i < rowSize; i++ {
			row[i] = 0
		}
		if _, err := f.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func putU32LE(b []byte, v uint32) {
	b[0] = byte(v); b[1] = byte(v >> 8); b[2] = byte(v >> 16); b[3] = byte(v >> 24)
}
