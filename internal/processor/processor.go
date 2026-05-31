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

// ProcessFolder scans dir for every target base name × every supported extension.
// It always returns a FolderResult — never panics or propagates errors upward.
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
				FileName: "unknown",
				Status:   models.StatusError,
				Message:  fmt.Sprintf("unexpected panic: %v", r),
			})
		}
	}()

	for _, baseName := range cfg.TargetBaseNames {
		logger.Debug(fmt.Sprintf("Folder=%q looking for base name %q", dir, baseName))
		results := processBaseName(dir, baseName, cfg, logger)
		result.ImageResults = append(result.ImageResults, results...)
	}

	// If no image results at all, every base name was missing.
	if len(result.ImageResults) == 0 {
		for _, baseName := range cfg.TargetBaseNames {
			result.ImageResults = append(result.ImageResults, &models.ImageResult{
				FileName: baseName + ".*",
				Status:   models.StatusTargetImageMissing,
				Message:  fmt.Sprintf("No supported image found for %q", baseName),
			})
		}
	}

	return result
}

// processBaseName finds all files matching baseName (any supported extension) in dir.
func processBaseName(dir, baseName string, cfg *config.Config, logger *logging.Logger) []*models.ImageResult {
	var results []*models.ImageResult

	for _, ext := range config.SupportedExtensions {
		candidate := filepath.Join(dir, baseName+ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			continue
		}

		logger.Debug(fmt.Sprintf("Found %q — processing", candidate))
		ir := processSingleImage(candidate, ext, cfg, logger)
		results = append(results, ir)
	}

	// Nothing found for this base name.
	if len(results) == 0 {
		results = append(results, &models.ImageResult{
			FileName: baseName + ".*",
			Status:   models.StatusTargetImageMissing,
			Message:  fmt.Sprintf("No supported image found for %q", baseName),
		})
	}

	return results
}

// processSingleImage loads, splits, and saves one image file.
func processSingleImage(sourcePath, ext string, cfg *config.Config, logger *logging.Logger) *models.ImageResult {
	fileName := filepath.Base(sourcePath)
	ir := &models.ImageResult{FileName: fileName}

	leftPath, rightPath := filesystem.OutputPaths(sourcePath, cfg.LeftSuffix, cfg.RightSuffix)

	if !cfg.OverwriteExisting && filesystem.ExistsAny(leftPath, rightPath) {
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

	bounds := img.Bounds()
	w := bounds.Max.X - bounds.Min.X
	h := bounds.Max.Y - bounds.Min.Y
	logger.Debug(fmt.Sprintf("File=%q dimensions=%dx%d splitAt=%d", fileName, w, h, w/2))

	left, right := splitVertically(img)

	if err := saveImage(left, leftPath, ext); err != nil {
		ir.Status = models.StatusError
		ir.Message = fmt.Sprintf("Could not save left image: %v", err)
		return ir
	}
	logger.Debug(fmt.Sprintf("Saved left: %q", leftPath))

	if err := saveImage(right, rightPath, ext); err != nil {
		ir.Status = models.StatusError
		ir.Message = fmt.Sprintf("Could not save right image: %v", err)
		return ir
	}
	logger.Debug(fmt.Sprintf("Saved right: %q", rightPath))

	if cfg.DeleteOriginal {
		if err := os.Remove(sourcePath); err != nil {
			ir.Status = models.StatusProcessed
			ir.Message = fmt.Sprintf("2 files created; warning: could not delete original: %v", err)
			return ir
		}
		logger.Debug(fmt.Sprintf("Deleted original: %q", sourcePath))
	}

	ir.Status = models.StatusProcessed
	ir.Message = fmt.Sprintf("2 files created (%dx%d → %dx%d + %dx%d)",
		w, h, w/2, h, w-w/2, h)
	return ir
}

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
	case ".bmp":
		return decodeBMP(f)
	case ".tiff", ".tif":
		return decodeTIFF(f)
	case ".webp":
		return decodeWEBP(f)
	default:
		return nil, fmt.Errorf("unsupported format: %s", ext)
	}
}

// splitVertically splits img into left and right halves.
// Odd widths: left=floor(w/2), right=ceil(w/2).
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
	case ".bmp":
		return encodeBMP(f, img)
	case ".tiff", ".tif":
		return encodeTIFF(f, img)
	case ".webp":
		// WebP encode: fall back to PNG to preserve lossless quality.
		return png.Encode(f, img)
	default:
		return fmt.Errorf("unsupported format: %s", ext)
	}
}

// ── BMP (pure stdlib, no dependency) ─────────────────────────────────────────

func decodeBMP(f *os.File) (image.Image, error) {
	// Use standard image package auto-detection.
	// BMP is supported via golang.org/x/image/bmp which Fyne already pulls in.
	// We re-use the registered decoder.
	f.Seek(0, 0)
	img, _, err := image.Decode(f)
	return img, err
}

func encodeBMP(f *os.File, img image.Image) error {
	// BMP encode: convert to PNG as BMP encoder isn't in stdlib.
	// Save as PNG with .bmp extension is not ideal; instead we write a raw BMP.
	b := img.Bounds()
	w := b.Max.X - b.Min.X
	h := b.Max.Y - b.Min.Y

	rowSize := ((w*3 + 3) / 4) * 4
	pixelDataSize := rowSize * h
	fileSize := 54 + pixelDataSize

	// BMP file header
	header := make([]byte, 54)
	header[0] = 'B'; header[1] = 'M'
	putU32LE(header[2:], uint32(fileSize))
	putU32LE(header[10:], 54)
	// DIB header
	putU32LE(header[14:], 40)
	putU32LE(header[18:], uint32(w))
	putU32LE(header[22:], uint32(h))
	header[26] = 1; header[27] = 0    // planes
	header[28] = 24; header[29] = 0   // bits per pixel
	putU32LE(header[34:], uint32(pixelDataSize))

	if _, err := f.Write(header); err != nil {
		return err
	}

	// BMP pixel data is stored bottom-up.
	row := make([]byte, rowSize)
	for y := h - 1; y >= 0; y-- {
		for x := 0; x < w; x++ {
			r, g, b2, _ := color.RGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).RGBA()
			row[x*3+0] = byte(b2 >> 8)
			row[x*3+1] = byte(g >> 8)
			row[x*3+2] = byte(r >> 8)
		}
		// Pad row to 4-byte boundary.
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

// ── TIFF (via golang.org/x/image — already a Fyne transitive dep) ────────────

func decodeTIFF(f *os.File) (image.Image, error) {
	f.Seek(0, 0)
	img, _, err := image.Decode(f)
	return img, err
}

func encodeTIFF(f *os.File, img image.Image) error {
	// Fall back to PNG for TIFF output (lossless, widely compatible).
	return png.Encode(f, img)
}

// ── WebP (via golang.org/x/image — already a Fyne transitive dep) ────────────

func decodeWEBP(f *os.File) (image.Image, error) {
	f.Seek(0, 0)
	img, _, err := image.Decode(f)
	return img, err
}
