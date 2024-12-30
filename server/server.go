package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DollarRate struct {
	ID   uint    `gorm:"primaryKey"`
	Rate float64 `gorm:"not null"`
	gorm.Model
}

func main() {

	// Configurar o banco de dados usando GORM
	db, err := gorm.Open(sqlite.Open("dollar_rates.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// Criar tabela
	err = db.AutoMigrate(&DollarRate{})
	if err != nil {
		panic(err)
	}

	// Definindo Endpoint /cotacao
	http.HandleFunc("/cotacao", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 200*time.Millisecond)
		defer cancel()

		rate, err := fetchDollarRate(ctx)

		if err != nil {
			// Verifica se o erro foi causado por timeout ou cancelamento
			if ctx.Err() == context.DeadlineExceeded {
				fmt.Println("requisição cancelada: timeout na busca da taxa de câmbio atingida!")
			} else {
				fmt.Println("erro ao fazer a requisição:", err)
			}

			http.Error(w, "falha ao buscar taxa de câmbio", http.StatusInternalServerError)
			return
		}

		dbCtx, dbCancel := context.WithTimeout(r.Context(), 10*time.Millisecond)
		defer dbCancel()

		err = db.WithContext(dbCtx).Create(&DollarRate{Rate: rate}).Error
		if err != nil {
			// Verifica se o erro foi causado por timeout ou cancelamento
			if dbCtx.Err() == context.DeadlineExceeded {
				fmt.Println("requisição cancelada: timeout na gravação do banco de dados atingido!")
			} else {
				fmt.Println("erro ao salvar cotação no banco de dados:", err)
			}

			http.Error(w, "falha ao buscar taxa de câmbio", http.StatusInternalServerError)
			return
		}

		response := map[string]float64{"bid": rate}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	http.ListenAndServe(":8080", nil)
}

func fetchDollarRate(ctx context.Context) (float64, error) {

	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return 0, err
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message := ""

		if m, e := io.ReadAll(resp.Body); e == nil {
			message = string(m)
		}
		return 0, fmt.Errorf("status code %d: %s", resp.StatusCode, message)
	}

	var result map[string]map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	rate, err := strconv.ParseFloat(result["USDBRL"]["bid"], 64)

	return rate, err
}
