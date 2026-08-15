package admin

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/HugoSmits86/nativewebp"
)

const (
	maxPrizeUploadSize  = 5 << 20
	maxPrizeImagePixels = 20_000_000
)

var prizeUploadDirectory = filepath.Join("static", "uploads", "prizes")

func savePrizeImage(file multipart.File, header *multipart.FileHeader, prizeName string) (string, string, error) {
	if header.Size <= 0 {
		return "", "", errors.New("file gambar kosong")
	}
	if header.Size > maxPrizeUploadSize {
		return "", "", errors.New("ukuran gambar maksimal 5 MB")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxPrizeUploadSize+1))
	if err != nil {
		return "", "", fmt.Errorf("membaca gambar: %w", err)
	}
	if len(data) > maxPrizeUploadSize {
		return "", "", errors.New("ukuran gambar maksimal 5 MB")
	}
	contentType := http.DetectContentType(data)
	if contentType != "image/jpeg" && contentType != "image/png" {
		return "", "", errors.New("format gambar harus JPG, JPEG, atau PNG")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return "", "", errors.New("file gambar tidak valid")
	}
	if int64(config.Width)*int64(config.Height) > maxPrizeImagePixels {
		return "", "", errors.New("resolusi gambar terlalu besar, maksimal 20 megapiksel")
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", "", errors.New("gambar tidak dapat dibaca")
	}

	randomPart := make([]byte, 4)
	if _, err = rand.Read(randomPart); err != nil {
		return "", "", fmt.Errorf("membuat nama gambar: %w", err)
	}
	filename := fmt.Sprintf("%s-%s-%s.webp", slugFilename(prizeName), time.Now().UTC().Format("20060102-150405"), hex.EncodeToString(randomPart))
	directory := prizeUploadDirectory
	if err = os.MkdirAll(directory, 0o755); err != nil {
		return "", "", fmt.Errorf("membuat direktori upload: %w", err)
	}
	finalPath := filepath.Join(directory, filename)
	temporary, err := os.CreateTemp(directory, ".prize-upload-*.tmp")
	if err != nil {
		return "", "", fmt.Errorf("membuat file gambar: %w", err)
	}
	temporaryPath := temporary.Name()
	cleanup := func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}
	if err = nativewebp.Encode(temporary, decoded, &nativewebp.Options{CompressionLevel: nativewebp.DefaultCompression}); err != nil {
		cleanup()
		return "", "", fmt.Errorf("mengonversi gambar ke WebP: %w", err)
	}
	if err = temporary.Close(); err != nil {
		cleanup()
		return "", "", fmt.Errorf("menyimpan gambar WebP: %w", err)
	}
	if err = os.Rename(temporaryPath, finalPath); err != nil {
		cleanup()
		return "", "", fmt.Errorf("menyelesaikan upload gambar: %w", err)
	}
	return "/static/uploads/prizes/" + filename, finalPath, nil
}

func slugFilename(value string) string {
	var result strings.Builder
	lastDash := false
	for _, char := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			if char <= unicode.MaxASCII {
				result.WriteRune(char)
				lastDash = false
			}
			continue
		}
		if result.Len() > 0 && !lastDash {
			result.WriteByte('-')
			lastDash = true
		}
	}
	name := strings.Trim(result.String(), "-")
	if name == "" {
		return "hadiah"
	}
	return name
}
