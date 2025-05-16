package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
)

type Column struct {
	Name string
	Type string
}

type AIResponse struct {
	SQL string `json:"sql"`
}

type PromptRequest struct {
	Prompt string `json:"prompt"`
}

var db *sql.DB
var schema string
var aiClient *openai.Client

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(".env dosyası okunamadı")
	}

	aiClient = openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Veritabanı bağlantı hatası:", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("Veritabanı ping hatası:", err)
	}

	tables, err := getTables(db)
	if err != nil {
		log.Fatal("Tablo çekme hatası:", err)
	}

	for _, table := range tables {
		schema += fmt.Sprintf("Table: %s\n", table)
		columns, err := getColumns(db, table)
		if err != nil {
			log.Printf("Kolon çekme hatası (%s): %v\n", table, err)
			continue
		}
		for _, col := range columns {
			schema += fmt.Sprintf(" - %s (%s)\n", col.Name, col.Type)
		}
		schema += "\n"
	}
	var endpoint_port = os.Getenv("ENDPOINT_PORT")
	http.HandleFunc("/getsql", handleSQLRequest)
	log.Printf("Sunucu http://localhost:%s adresinde başlatıldı...\n", endpoint_port)
	log.Fatal(http.ListenAndServe(":"+endpoint_port, nil))
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func handleSQLRequest(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Sadece POST isteği destekleniyor", http.StatusMethodNotAllowed)
		return
	}

	var req PromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Geçersiz JSON formatı", http.StatusBadRequest)
		return
	}

	result, err := generateSQL(req.Prompt)
	if err != nil {
		http.Error(w, fmt.Sprintf("AI hatası: %v", err), http.StatusInternalServerError)
		return
	}

	respBytes, _ := json.MarshalIndent(result, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

func generateSQL(userPrompt string) (AIResponse, error) {
	resp, err := aiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: "system",
					Content: `Sen bir SQL uzmanısın. Aşağıdaki MySQL şemasına göre gelen isteğe uygun optimize SQL sorgusu üret.
Cevabı sadece aşağıdaki JSON formatında döndür (önüne veya sonuna markdown işareti koyma):
{
  "sql": "SQL SORGUSU"
}`,
				},
				{
					Role:    "user",
					Content: fmt.Sprintf("Veritabanı Şeması:\n%s\n\nSORU: %s", schema, userPrompt),
				},
			},
		},
	)
	if err != nil {
		return AIResponse{}, err
	}

	raw := strings.TrimSpace(resp.Choices[0].Message.Content)
	clean := stripMarkdownJSON(raw)

	var result AIResponse
	if err := json.Unmarshal([]byte(clean), &result); err != nil {
		return AIResponse{}, fmt.Errorf("OpenAI çıktısı geçersiz JSON: %v\nCevap:\n%s", err, raw)
	}

	return result, nil
}

func getTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, nil
}

func getColumns(db *sql.DB, table string) ([]Column, error) {
	query := fmt.Sprintf("SHOW COLUMNS FROM `%s`", table)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []Column
	for rows.Next() {
		var field, colType, null, key, extra string
		var def sql.NullString

		if err := rows.Scan(&field, &colType, &null, &key, &def, &extra); err != nil {
			return nil, err
		}
		cols = append(cols, Column{Name: field, Type: colType})
	}
	return cols, nil
}

func stripMarkdownJSON(s string) string {
	re := regexp.MustCompile("(?s)```json(.*?)```")
	matches := re.FindStringSubmatch(s)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return s
}
