package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"
)

type ExchangeRate struct {
	Bid float64
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	rate, err := fetchExchangeRate(ctx)
	if err != nil {
		// Verifica se o erro foi causado por timeout ou cancelamento
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Println("requisição cancelada: timeout na busca da taxa de câmbio atingida!")
		}
		log.Fatalf("erro ao fazer a requisição: %v", err)
	}

	err = saveToFile(rate)
	if err != nil {
		log.Fatalf("falha ao salvar cotação no arquivo: %v", err)
	}

	fmt.Println("cotação salvou com sucesso!")
}

func fetchExchangeRate(ctx context.Context) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
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
		return 0, fmt.Errorf("status code %d", resp.StatusCode)
	}

	var result map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result["bid"], nil
}

func saveToFile(rate float64) error {
	const templateText = "Dólar: {{.Bid}}\n"

	// Abrir ou criar o arquivo cotacao.txt no modo de append
	file, err := os.OpenFile("cotacao.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("falha ao abrir arquivo: %v", err)
	}
	defer file.Close()

	// Criar e executar o template
	tmpl, err := template.New("cotacao").Parse(templateText)
	if err != nil {
		return fmt.Errorf("falha ao criar template: %v", err)
	}

	data := ExchangeRate{Bid: rate}
	return tmpl.Execute(file, data)
}
