package converter

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"
)

// IsProcessed checks if the video has already been processed successfully
func IsProcessed(db *sql.DB, videoID int) bool {
	var IsProcessed bool
	query := "SELECT EXISTS(SELECT 1 FROM processed_videos where video_id = $1 and status='success')"
	err := db.QueryRow(query, videoID).Scan(&IsProcessed)
	if err != nil {
		slog.Error("Error checking if video is processed", slog.Int("video_id", videoID))
		return false
	}
	return IsProcessed
}

// MarkProcessed registers that the video has been processed successfully
func MarkProcess(db *sql.DB, videoID int) error {
	query := "INSERT INTO processed_videos (video_id, status, processed_at) values ($1, $2, $3)"
	_, err := db.Exec(query, videoID, "success", time.Now())
	if err != nil {
		slog.Error("Error marking video as processed", slog.Int("video_id", videoID))
		return err
	}
	return nil
}

// RegisterError stores the error details and phase history in the database
func RegisterError(db *sql.DB, errorData map[string]interface{}, err error) {
	serializedError, _ := json.Marshal(errorData)
	query := "INSERT INTO process_errors_log (error_details, created_at) VALUES ($1, $2)"
	_, dbErr := db.Exec(query, serializedError, time.Now())
	if dbErr != nil {
		slog.Error("Error storing error log in database", slog.String("error", dbErr.Error()))
		return
	}
	slog.Info("Error log stored successfully", slog.String("error", err.Error()))
}