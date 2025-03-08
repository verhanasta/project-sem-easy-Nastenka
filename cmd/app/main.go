package main

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type StorageConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

type PriceRecord struct {
	ID         int
	ItemName   string
	Group      string
	Cost       float64
	RecordTime string
}

var storage *sql.DB

const (
	defaultPort   = 8080
	maxFileSize   = 10 << 20 // 10MB
	archiveFormat = "zip"
	dataFileName  = "export.csv"
)

func init() {
	log.SetOutput(os.Stdout)
}

func main() {
	cfg := StorageConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "validator",
		Password: "val1dat0r",
		Database: "project-sem-1",
	}

	conn, err := setupStorage(cfg)
	if err != nil {
		log.Fatalf("Storage initialization failed: %v", err)
	}
	defer conn.Close()
	storage = conn

	if err = prepareStorage(); err != nil {
		log.Fatalf("Storage preparation failed: %v", err)
	}

	router := http.NewServeMux()
	router.HandleFunc("/api/v0/prices", processRequests)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", defaultPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("Starting service on port %d", defaultPort)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failure: %v", err)
	}
}

func setupStorage(cfg StorageConfig) (*sql.DB, error) {
	connString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database,
	)
	conn, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("connection error: %w", err)
	}

	if err = conn.Ping(); err != nil {
		return nil, fmt.Errorf("connection test failed: %w", err)
	}
	return conn, nil
}

func prepareStorage() error {
	_, err := storage.Exec(`CREATE TABLE IF NOT EXISTS prices (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
		price NUMERIC(10,2) NOT NULL,
		create_date TIMESTAMP NOT NULL
	)`)
	return err
}

func processRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handleDataUpload(w, r)
	case http.MethodGet:
		handleDataExport(w, r)
	default:
		respondWithError(w, "Unsupported method", http.StatusMethodNotAllowed)
	}
}

func handleDataUpload(w http.ResponseWriter, r *http.Request) {
	file, info, err := r.FormFile("file")
	if err != nil {
		respondWithError(w, "Invalid file upload", http.StatusBadRequest)
		return
	}
	defer file.Close()

	log.Printf("Processing archive: %s (size: %d)", info.Filename, info.Size)

	archiveData, err := io.ReadAll(io.LimitReader(file, maxFileSize))
	if err != nil {
		log.Printf("Archive read error: %v", err)
		respondWithError(w, "Archive processing failed", http.StatusInternalServerError)
		return
	}

	content, err := extractFromArchive(archiveData)
	if err != nil {
		log.Printf("Archive extraction error: %v", err)
		respondWithError(w, err.Error(), http.StatusBadRequest)
		return
	}

	validRecords, err := validateCSVContent(content)
	if err != nil {
		respondWithError(w, "CSV validation failed", http.StatusBadRequest)
		return
	}

	inserted, err := saveRecords(validRecords)
	if err != nil {
		log.Printf("Data save error: %v", err)
		respondWithError(w, "Data storage failure", http.StatusInternalServerError)
		return
	}

	stats, err := calculateStatistics()
	if err != nil {
		log.Printf("Statistics error: %v", err)
		respondWithError(w, "Statistics calculation failed", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"processed_items": inserted,
		"category_count":  stats.Categories,
		"total_amount":    stats.Total,
	})
}

func extractFromArchive(data []byte) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid archive format")
	}

	for _, f := range reader.File {
		if strings.HasSuffix(f.Name, ".csv") {
			file, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("cannot open archive file")
			}
			defer file.Close()

			content, err := io.ReadAll(file)
			if err != nil {
				return nil, fmt.Errorf("archive content read failed")
			}
			return content, nil
		}
	}
	return nil, errors.New("CSV file not found in archive")
}

func validateCSVContent(data []byte) ([]PriceRecord, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = ','
	lines, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV parsing error")
	}

	var results []PriceRecord
	for idx, line := range lines {
		if idx == 0 || len(line) != 5 {
			continue
		}

		id, _ := strconv.Atoi(line[0])
		cost, err := strconv.ParseFloat(line[3], 64)
		if err != nil {
			continue
		}

		results = append(results, PriceRecord{
			ID:         id,
			ItemName:   line[1],
			Group:      line[2],
			Cost:       cost,
			RecordTime: line[4],
		})
	}

	if len(results) == 0 {
		return nil, errors.New("no valid records found")
	}
	return results, nil
}

func saveRecords(records []PriceRecord) (int, error) {
	tx, err := storage.Begin()
	if err != nil {
		return 0, fmt.Errorf("transaction start failed")
	}

	stmt, err := tx.Prepare(`INSERT INTO prices 
	(name, category, price, create_date)
	VALUES ($1, $2, $3, $4)`)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("statement preparation failed")
	}
	defer stmt.Close()

	var count int
	for _, r := range records {
		_, err = stmt.Exec(r.ItemName, r.Group, r.Cost, r.RecordTime)
		if err == nil {
			count++
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("transaction commit failed")
	}
	return count, nil
}

type Statistics struct {
	Categories int
	Total      float64
}

func calculateStatistics() (*Statistics, error) {
	var stats Statistics
	err := storage.QueryRow(`
	SELECT COUNT(DISTINCT category), SUM(price) 
	FROM prices`).Scan(&stats.Categories, &stats.Total)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func handleDataExport(w http.ResponseWriter, r *http.Request) {
	records, err := fetchAllRecords()
	if err != nil {
		respondWithError(w, "Data retrieval failed", http.StatusInternalServerError)
		return
	}

	archive, err := createExportArchive(records)
	if err != nil {
		respondWithError(w, "Archive creation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=export.zip")
	w.Header().Set("Content-Length", strconv.Itoa(len(archive)))
	w.Write(archive)
}

func fetchAllRecords() ([]PriceRecord, error) {
	rows, err := storage.Query(`
	SELECT id, name, category, price, create_date
	FROM prices`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []PriceRecord
	for rows.Next() {
		var r PriceRecord
		err := rows.Scan(&r.ID, &r.ItemName, &r.Group, &r.Cost, &r.RecordTime)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func createExportArchive(records []PriceRecord) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	headers := []string{"ID", "Product", "Category", "Price", "Timestamp"}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	for _, r := range records {
		row := []string{
			strconv.Itoa(r.ID),
			r.ItemName,
			r.Group,
			fmt.Sprintf("%.2f", r.Cost),
			r.RecordTime,
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}
	writer.Flush()

	var zipBuf bytes.Buffer
	archive := zip.NewWriter(&zipBuf)
	file, _ := archive.Create(dataFileName)
	file.Write(buf.Bytes())
	archive.Close()

	return zipBuf.Bytes(), nil
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, message string, code int) {
	respondWithJSON(w, code, map[string]string{"error": message})
}
