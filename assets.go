package main

import (
	"os"
	"fmt"
	"strings"
	"path/filepath"
	"crypto/rand"
	"encoding/base64"
)

func generateToken() string {
	key := make([]byte, 32)
	rand.Read(key)
	return base64.RawURLEncoding.EncodeToString(key)
}

func extractExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func (cfg *apiConfig) getAssetPath(assetName, mediaType string) string {
	ext := extractExt(mediaType)
	return fmt.Sprintf("%s%s", assetName, ext)
}

func (cfg *apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func (cfg *apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://%s:%s/assets/%s", cfg.host, cfg.port, assetPath)
}


