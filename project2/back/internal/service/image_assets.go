package service

import (
	"bytes"
	"image"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

type preparedImageAssets struct {
	Original []byte
	Preview  []byte
	Thumb    []byte
	Ext      string
}

func buildPreparedImageAssets(
	originalBytes []byte,
	fileName string,
	imageResult map[string]any,
	fit func(image.Image, int) image.Image,
	originalQuality int,
	previewQuality int,
	thumbQuality int,
	encode func(image.Image, int) ([]byte, error),
) (preparedImageAssets, error) {
	img, _, err := image.Decode(bytes.NewReader(originalBytes))
	if err == nil {
		originalSan := fit(img, 2048)
		preview := fit(img, 1200)
		thumb := fit(img, 800)

		origJPG, err := encode(originalSan, originalQuality)
		if err != nil {
			return preparedImageAssets{}, err
		}
		prevJPG, err := encode(preview, previewQuality)
		if err != nil {
			return preparedImageAssets{}, err
		}
		thumbJPG, err := encode(thumb, thumbQuality)
		if err != nil {
			return preparedImageAssets{}, err
		}

		return preparedImageAssets{
			Original: origJPG,
			Preview:  prevJPG,
			Thumb:    thumbJPG,
			Ext:      ".jpg",
		}, nil
	}

	if canUseRawImageFallback(originalBytes, fileName, imageResult) {
		ext := preferredImageExtension(originalBytes, fileName)
		log.Printf("[ИЗОБРАЖЕНИЕ] Использован запасной сценарий сохранения без пересборки превью: файл=%q ext=%q type=%q result=%v", fileName, ext, strings.ToLower(http.DetectContentType(originalBytes)), imageResult)
		return preparedImageAssets{
			Original: originalBytes,
			Preview:  originalBytes,
			Thumb:    originalBytes,
			Ext:      ext,
		}, nil
	}

	return preparedImageAssets{}, err
}

func looksLikeWebP(originalBytes []byte, fileName string) bool {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == ".webp" {
		return true
	}

	mimeByExt := strings.ToLower(mime.TypeByExtension(ext))
	if strings.HasPrefix(mimeByExt, "image/webp") {
		return true
	}

	contentType := strings.ToLower(http.DetectContentType(originalBytes))
	if strings.HasPrefix(contentType, "image/webp") {
		return true
	}

	return len(originalBytes) >= 12 &&
		string(originalBytes[:4]) == "RIFF" &&
		string(originalBytes[8:12]) == "WEBP"
}

func canUseRawImageFallback(originalBytes []byte, fileName string, imageResult map[string]any) bool {
	if looksLikeWebP(originalBytes, fileName) {
		return true
	}

	contentType := strings.ToLower(http.DetectContentType(originalBytes))
	if strings.HasPrefix(contentType, "image/") {
		return true
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp", ".avif", ".heic", ".heif":
		return true
	}

	if imageResult == nil {
		return false
	}
	if unavailable, ok := imageResult["unavailable"].(bool); ok && unavailable {
		return false
	}
	if status, ok := imageResult["status"].(string); ok && strings.TrimSpace(status) != "" {
		return true
	}

	return false
}

func preferredImageExtension(originalBytes []byte, fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp", ".avif", ".heic", ".heif":
		return ext
	}

	contentType := strings.ToLower(http.DetectContentType(originalBytes))
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/bmp":
		return ".bmp"
	case "image/avif":
		return ".avif"
	case "image/heic", "image/heif":
		return ".heic"
	default:
		return ".img"
	}
}
