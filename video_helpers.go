package main

import (
	"fmt"
	"bytes"
	"os/exec"
	"encoding/json"
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

