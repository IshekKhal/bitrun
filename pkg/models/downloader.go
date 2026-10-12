package models

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ProgressWriter tracks bytes written and displays progress.
type ProgressWriter struct {
	Total      int64
	Downloaded int64
	StartTime  time.Time
	LastPrint  time.Time
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.Downloaded += int64(n)

	now := time.Now()
	if now.Sub(pw.LastPrint) > 250*time.Millisecond || pw.Downloaded == pw.Total {
		pw.LastPrint = now
		elapsed := now.Sub(pw.StartTime).Seconds()
		speedMB := float64(pw.Downloaded) / (1024 * 1024) / elapsed
		percent := float64(0)
		if pw.Total > 0 {
			percent = float64(pw.Downloaded) / float64(pw.Total) * 100
		}
		fmt.Printf("\r[bitrun] Downloading: %.1f%% (%.1f MB / %.1f MB) at %.1f MB/s   ",
			percent,
			float64(pw.Downloaded)/(1024*1024),
			float64(pw.Total)/(1024*1024),
			speedMB,
		)
	}
	return n, nil
}

// DownloadModel fetches the specified model weights from the remote repository.
func DownloadModel(def ModelDefinition, targetDir string) (string, error) {
	targetFile := filepath.Join(targetDir, def.Filename)

	// Check if already present
	if info, err := os.Stat(targetFile); err == nil && info.Size() > 0 {
		fmt.Printf("[bitrun] Model %s already cached at %s\n", def.Tag, targetFile)
		return targetFile, nil
	}

	remoteFile := def.RemoteFilename
	if remoteFile == "" {
		remoteFile = def.Filename
	}
	url := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", def.Repo, remoteFile)
	fmt.Printf("[bitrun] Fetching %s from %s...\n", def.Tag, url)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to initiate download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: failed to download from %s", resp.StatusCode, url)
	}

	tmpFile := targetFile + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer out.Close()

	pw := &ProgressWriter{
		Total:     resp.ContentLength,
		StartTime: time.Now(),
		LastPrint: time.Now(),
	}

	_, err = io.Copy(out, io.TeeReader(resp.Body, pw))
	if err != nil {
		os.Remove(tmpFile)
		return "", fmt.Errorf("download interrupted: %w", err)
	}
	fmt.Println()

	if err := os.Rename(tmpFile, targetFile); err != nil {
		return "", fmt.Errorf("failed to finalize model file: %w", err)
	}

	fmt.Printf("[bitrun] Successfully installed %s (%.1f MB)\n", def.Tag, float64(resp.ContentLength)/(1024*1024))
	return targetFile, nil
}
