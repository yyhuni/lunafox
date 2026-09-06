package enginepackagebuild

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"time"
)

func writeDeterministicTarGzipInOrder(writer io.Writer, entries map[string][]byte, paths []string) error {
	for _, path := range paths {
		if _, exists := entries[path]; !exists {
			return fmt.Errorf("engine package archive entry %q is missing", path)
		}
	}
	gz, err := gzip.NewWriterLevel(writer, gzip.BestCompression)
	if err != nil {
		return err
	}
	gz.Name = ""
	gz.Comment = ""
	gz.Extra = nil
	gz.ModTime = time.Unix(0, 0)
	gz.OS = 255
	tw := tar.NewWriter(gz)
	for _, entryPath := range paths {
		payload := entries[entryPath]
		header := &tar.Header{
			Name:     entryPath,
			Mode:     0o644,
			Size:     int64(len(payload)),
			ModTime:  time.Unix(0, 0),
			Typeflag: tar.TypeReg,
			Format:   tar.FormatUSTAR,
		}
		if err := tw.WriteHeader(header); err != nil {
			_ = tw.Close()
			_ = gz.Close()
			return err
		}
		if _, err := tw.Write(payload); err != nil {
			_ = tw.Close()
			_ = gz.Close()
			return err
		}
	}
	if err := tw.Close(); err != nil {
		_ = gz.Close()
		return err
	}
	return gz.Close()
}
