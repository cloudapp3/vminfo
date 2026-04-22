package updater

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ExtractBinaryFromTarGz extracts the binary named binaryName from a .tar.gz
// archive, writing it to destDir and returning the path to the extracted file.
func ExtractBinaryFromTarGz(archivePath, binaryName, destDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		base := filepath.Base(hdr.Name)
		if base != binaryName {
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		outPath := filepath.Join(destDir, binaryName)
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return "", err
		}
		out.Close()
		return outPath, nil
	}

	return "", fmt.Errorf("binary %q not found in archive", binaryName)
}
