package handlers

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"project-sem/internal/db"
	"strings"
)

// HandlerGetPrices обрабатывает GET-запрос для получения данных из базы данных
func HandlerPostPrices() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB
			http.Error(w, "Error parsing form: "+err.Error(), http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "File upload error: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Читаем содержимое файла в память
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "File read error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Создаем zip.Reader из байтов
		zipReader, err := zip.NewReader(strings.NewReader(string(fileBytes)), int64(len(fileBytes)))
		if err != nil {
			http.Error(w, "ZIP read error: "+err.Error(), http.StatusBadRequest)
			return
		}

		var csvFile *zip.File
		for _, f := range zipReader.File {
			if strings.HasSuffix(f.Name, ".csv") {
				csvFile = f
				break
			}
		}
		if csvFile == nil {
			http.Error(w, "CSV file not found", http.StatusBadRequest)
			return
		}

		rc, err := csvFile.Open()
		if err != nil {
			http.Error(w, "CSV open error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer rc.Close()

		reader := csv.NewReader(rc)
		records, err := reader.ReadAll()
		if err != nil {
			http.Error(w, "CSV parse error: "+err.Error(), http.StatusBadRequest)
			return
		}

		var inputPrices []db.InputPrice
		for i := 1; i < len(records); i++ {
			price, err := db.ParseInputPrice(records[i])
			if err != nil {
				http.Error(w, "Data parsing error: "+err.Error(), http.StatusBadRequest)
				return
			}
			inputPrices = append(inputPrices, price)
		}

		totalItems, totalCategories, totalPrice, err := db.InsertPrices(inputPrices)
		if err != nil {
			http.Error(w, "DB insert error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"total_items":      totalItems,
			"total_categories": totalCategories,
			"total_price":      totalPrice,
		}

		w.Header().Set("Content-Type", "application/json")
		if err = json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "JSON encode error: "+err.Error(), http.StatusInternalServerError)
		}
	}
}