package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// VerifySHA256 verifies that the file at filePath matches the expected hex checksum.
func VerifySHA256(filePath, expectedHex string) (bool, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	return actual == expectedHex, nil
}
