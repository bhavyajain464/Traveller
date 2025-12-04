package utils

import (
	"fmt"
	"image/png"
	"os"

	"github.com/skip2/go-qrcode"
)

// GenerateQRCodeImage generates a PNG image of the QR code
func GenerateQRCodeImage(qrCode string, size int) (string, error) {
	if size <= 0 {
		size = 256
	}

	// Generate QR code
	qr, err := qrcode.New(qrCode, qrcode.Medium)
	if err != nil {
		return "", fmt.Errorf("failed to generate QR code: %w", err)
	}

	// Create file path
	filename := fmt.Sprintf("/tmp/qr_%s.png", qrCode)
	file, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Encode as PNG
	err = png.Encode(file, qr.Image(size))
	if err != nil {
		return "", fmt.Errorf("failed to encode PNG: %w", err)
	}

	return filename, nil
}

// GenerateQRCodeBase64 generates QR code as base64 encoded string
func GenerateQRCodeBase64(qrCode string, size int) (string, error) {
	if size <= 0 {
		size = 256
	}

	png, err := qrcode.Encode(qrCode, qrcode.Medium, size)
	if err != nil {
		return "", fmt.Errorf("failed to generate QR code: %w", err)
	}

	return string(png), nil
}


