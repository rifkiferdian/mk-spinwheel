package admin

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HugoSmits86/nativewebp"
)

func TestSavePrizeImageConvertsPNGToNamedWebP(t *testing.T) {
	oldDirectory := prizeUploadDirectory
	prizeUploadDirectory = t.TempDir()
	defer func() { prizeUploadDirectory = oldDirectory }()

	input := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			input.Set(x, y, color.NRGBA{R: 249, G: 115, B: 22, A: 255})
		}
	}
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, input); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("image", "Hadiah Asli.PNG")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(pngData.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err = request.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	file, header, err := request.FormFile("image")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	publicPath, diskPath, err := savePrizeImage(file, header, "Voucher Rp10.000")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(filepath.Base(diskPath), "voucher-rp10-000-") || filepath.Ext(diskPath) != ".webp" {
		t.Fatalf("nama otomatis tidak valid: %s", diskPath)
	}
	if !strings.HasPrefix(publicPath, "/static/uploads/prizes/") {
		t.Fatalf("path publik tidak valid: %s", publicPath)
	}
	stored, err := os.Open(diskPath)
	if err != nil {
		t.Fatal(err)
	}
	defer stored.Close()
	decoded, err := nativewebp.Decode(stored)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Dx() != 8 || decoded.Bounds().Dy() != 8 {
		t.Fatalf("ukuran WebP berubah: %v", decoded.Bounds())
	}
}
