package main

import (
	"os"
	"io"
	"fmt"
	"mime"
	"strings"
	"slices"
	"net/http"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)


func extractFileExtension(contentType string, options []string) (string, error) {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", err
	}

	if !slices.Contains(options, mediaType) {
		return "", fmt.Errorf("unsupported media type")
	}

	ext := "." + mediaType[strings.Index(mediaType, "/") + 1:]
	return ext, nil
}


func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	formFile, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer formFile.Close()
	
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Video not found", err)
		return
	}

	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "This is not your video >:(", err)
		return
	}

	contentType := header.Header.Get("Content-Type")
	allowedTypes := []string{ "image/jpeg", "image/png" }
	ext, err := extractFileExtension(contentType, allowedTypes)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid filetype", err)
		return
	}

	videoFilepath := filepath.Join(cfg.assetsRoot, videoIDString + ext)

	videoFile, err := os.Create(videoFilepath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create video file", err)
		return
	}

	if _, err = io.Copy(videoFile, formFile); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to parse form file", err)
		return
	}

	thumbnailURL := "http://localhost:" + cfg.port + "/" + videoFilepath
	video.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
