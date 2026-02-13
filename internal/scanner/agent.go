package scanner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type Agent struct {
	Dirs       []string
	APIBaseURL string
	APIKey     string
	DB         *bbolt.DB
	Log        *zap.Logger
	Limit      *rate.Limiter
	Workers    int
}

func (a *Agent) RunOnce() error {
	for _, dir := range a.Dirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			return a.sendFile(path)
		})
		if err != nil {
			return fmt.Errorf("walk dir: %w", err)
		}
	}
	return nil
}

func (a *Agent) sendFile(path string) error {
	if err := a.Limit.WaitN(context.Background(), 1); err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	stat, _ := f.Stat()
	date := stat.ModTime().Format("2006-01-02")
	virtualPath := filepath.ToSlash(path)
	url := fmt.Sprintf("%s/v1/upload?date=%s&path=%s", a.APIBaseURL, date, virtualPath)
	req, _ := http.NewRequest(http.MethodPost, url, f)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-API-Key", a.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed: %s", bytes.TrimSpace(b))
	}
	a.Log.Info("uploaded", zap.String("path", path))
	return a.markSent(path, stat)
}

func (a *Agent) markSent(path string, st os.FileInfo) error {
	return a.DB.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("files"))
		if err != nil {
			return err
		}
		return b.Put([]byte(path), []byte(fmt.Sprintf("%d|%d|%s", st.Size(), st.ModTime().Unix(), time.Now().Format(time.RFC3339))))
	})
}
