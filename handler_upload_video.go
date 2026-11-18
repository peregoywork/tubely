package main

import (
	"os"
	"io"
	"fmt"
	"mime"
	"net/http"

	"github.com/google/uuid"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
)



func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10 << 30) // 1 GB

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

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't find video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorized to update this video", nil)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", nil)
		return
	}

	tempDst, err := os.CreateTemp("", "tubely-upload.mp4")
	defer tempDst.Close()
	defer os.Remove(tempDst.Name())

	if _, err = io.Copy(tempDst, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error: failed to copy from memory to disk", err)
		return
	}
	
	aspectRatio, err := getVideoAspectRatio(tempDst.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error: failed to get aspect ratio", err)
		return
	}

	fastFilePath, err := processVideoForFastStart(tempDst.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error: failed to make 'faststart' video", err)
		return
	}

	fastFile, err := os.Open(fastFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error: failed to open file", err)
		return
	}
	defer fastFile.Close()
	defer os.Remove(fastFile.Name())

	fastFile.Seek(0, io.SeekStart)
	key := fmt.Sprintf("%s/%s.%s", aspectRatio, generateToken(), extractExt(mediaType))

	_, err = uploadToBucket(r.Context(), cfg.s3Client, cfg.s3Bucket, key, mediaType, fastFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to upload to s3 bucket", err)
		return
	}

	// s3URL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key)
	hackyURL := fmt.Sprintf("%s/%s", cfg.s3Bucket, key)
	video.VideoURL = &hackyURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "database failed to update video entry", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
