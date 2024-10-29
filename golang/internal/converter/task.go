package converter

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"
)

// VideoConverter handles video conversion tasks
type VideoConverter struct{}

// NewVideoConverter creates a new instance of VideoConverter
func NewVideoConverter() *VideoConverter {
	return &VideoConverter{}
}

// VideoTask represents a video conversion task
type VideoTask struct {
	VideoID int    `json:"video_id"`
	Path    string `json:"path"`
}

// HandlerMessage processes a video conversion message
func (vc *VideoConverter) Handle(msg []byte) {
	var task VideoTask

	err := json.Unmarshal(msg, &task)
	if err != nil {
		vc.logError(task, "failed to unmarshal task", err)
		return
	}

	// Process the video
	err = vc.processVideo(&task)
	if err != nil {
		vc.logError(task, "failed to process video", err)
		return
	}
}

// processVideo handles video processing (merging chunks and converting)
func (vc *VideoConverter) processVideo(task *VideoTask) error {
	mergedFile := filepath.Join(task.Path, "merged.mp4")
	mpegDashPath := filepath.Join(task.Path, "mpeg-dash")

	// Merge chunks
	slog.Info("Merging chunks", slog.String("path", task.Path))
	err := vc.mergeChunks(task.Path, mergedFile)
	if err != nil {
		vc.logError(*task, "failed to merge chunks", err)
		return err
	}

	// Create directory for MPEG-DASH output
	slog.Info("Creating mpeg-dash dir", slog.String("path", task.Path))
	err = os.MkdirAll(mpegDashPath, os.ModePerm)
	if err != nil {
		vc.logError(*task, "failed to create mpeg-dash directory", err)
		return err
	}

	// Convert to MPEG-DASH
	slog.Info("Converting video to mpeg-dash", slog.String("path", task.Path))
	ffmpegCmd := exec.Command(
		"ffmpeg", "-i", mergedFile, //Arquivo de entrada
		"-f", "dash", // Formato de saída
		filepath.Join(mpegDashPath, "output.mpd"), // Caminho para salvar o arquivo .mpd
	)

	output, err := ffmpegCmd.CombinedOutput()
	if err != nil {
		vc.logError(*task, "failed to convert video to mpeg-dash, output: "+string(output), err)
		return err
	}
	slog.Info("Video convert to mpeg-dash", slog.String("path", mpegDashPath))


	//Remove merged file after processing
	slog.Info("Removing merged file", slog.String("path", mergedFile))
	err = os.Remove(mergedFile)
	if err != nil {
		vc.logError(*task, "failed to remove merged file", err)
		return err
	}
	return nil
}

// logError handles logging the error in JSON format
func (vc *VideoConverter) logError(task VideoTask, message string, err error) {
	errorData := map[string]any{
		"video_id": task.VideoID,
		"error":    message,
		"details":  err.Error(),
		"time":     time.Now(),
	}
	serializedError, _ := json.Marshal(errorData)
	slog.Error("Processing error", slog.String("error_details", string(serializedError)))

	// todo: register error on database
}

// Método para extrair o número do nome do arquivo
func (vc *VideoConverter) extractNumber(fileName string) int {
	re := regexp.MustCompile(`\d+`)
	numStr := re.FindString(filepath.Base(fileName)) // Pega o nome do arquivo, sem o caminho
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return -1
	}
	return num
}

// Método para mesclar os chunks
func (vc *VideoConverter) mergeChunks(inputDir, outputFile string) error {
	// Buscar todos os arquivos .chunk no diretório
	chunks, err := filepath.Glob(filepath.Join(inputDir, "*.chunk"))
	if err != nil {
		return fmt.Errorf("failed to find chunks: %v", err)
	}

	// Ordenar os chunks numericamente
	sort.Slice(chunks, func(i, j int) bool {
		return vc.extractNumber(chunks[i]) < vc.extractNumber(chunks[j])
	})

	// Criar arquivo de saída
	output, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer output.Close()

	// Ler cada chunk e escrever no arquivo final
	for _, chunk := range chunks {
		input, err := os.Open(chunk)
		if err != nil {
			return fmt.Errorf("failed to open chunk: %v", err)
		}

		// Copiar dados do chunk para o arquivo de saída
		_, err = output.ReadFrom(input)
		if err != nil {
			return fmt.Errorf("failed to write chunk %s to merged file: %v", chunk, err)
		}
		input.Close()
	}
	return nil
}
