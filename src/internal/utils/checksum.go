package utils

import (
	"crypto/sha256"
	"fmt"
	"hash/crc32"
	"io"
	"os"
)

// CalculateFileChecksum calculates the checksum for a file using CRC32 algorithm
func CalculateFileCRC32Checksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := crc32.New(crc32.MakeTable(crc32.IEEE))
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Convert to hex string for storage
	checksum := fmt.Sprintf("%08x", hash.Sum32())
	return checksum, nil
}

// CalculateFileSHA256Checksum calculates the checksum for a file using SHA256 algorithm
func CalculateFileSHA256Checksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Convert to hex string for storage
	checksum := fmt.Sprintf("%x", hash.Sum(nil))
	return checksum, nil
}

// VerifyFileChecksum checks if a file's checksum matches the expected value
func VerifyFileChecksum(filePath string, expectedChecksum string) (bool, error) {
	calculatedChecksum, err := CalculateFileCRC32Checksum(filePath)
	if err != nil {
		return false, err
	}

	return calculatedChecksum == expectedChecksum, nil
}

// CalculateFileChecksumUsingAlgorithm calculates file checksum using a specified algorithm
func CalculateFileChecksumUsingAlgorithm(filePath string, algorithm string) (string, error) {
	switch algorithm {
	case "crc32":
		return CalculateFileCRC32Checksum(filePath)
	case "sha256":
		return CalculateFileSHA256Checksum(filePath)
	default:
		return CalculateFileCRC32Checksum(filePath) // Default to CRC32
	}
}