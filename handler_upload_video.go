package main

import (
	"os"
	"io"
	"fmt"
	"mime"
	"bytes"
	"os/exec"
	"net/http"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	var data map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &data); err != nil {
		return "", err
	}

	streamData, ok := data["streams"].([]any)
	if !ok {
		return "", fmt.Errorf("'streams' key missing or did not point to array")
	}
	firstStream, ok := streamData[0].(map[string]any) 
	if !ok {
		return "", fmt.Errorf("first stream element is not an object")
	}
	ratio, ok := firstStream["display_aspect_ratio"]
	if !ok {
		return "", fmt.Errorf("'display_aspect_ratio' key missing")
	}

	if ratio == "16:9" { return "landscape", nil }
	if ratio == "9:16" { return "portrait", nil }
	return "other", nil
}

func processVideoForFastStart(filePath string) (string, error) {
	procFilePath := fmt.Sprintf("%s.processing", filePath)
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", procFilePath)
	
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return procFilePath, nil
}

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
	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket: &cfg.s3Bucket,
		Key: &key,
		Body: fastFile,
		ContentType: &mediaType,
	})

	s3URL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key)
	video.VideoURL = &s3URL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error: database failed to update video entry", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
