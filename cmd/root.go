package cmd

import (
	"encoding/json"
	"io"
	"load-tests-cli/internal/entity"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var finalReport entity.Report

var rootCmd = &cobra.Command{
	Use:   "load-tests-cli",
	Short: "Realizar testes de carga em serviços web",
	Long: `O usuário deverá fornecer a URL do serviço, o número total de requisições e a quantidade de chamadas simultâneas.
	Ao final, o sistema deve gerar um relatório detalhado da execução`,
	RunE: func(cmd *cobra.Command, args []string) error {
		report, err := getLoads()
		if err != nil {
			return err
		}

		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	},
}

func getLoads() (*entity.Report, error) {
	return runLoad(http.DefaultClient, URL, TotalRequests, Concurrency)
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type requestResult struct {
	status  int
	success bool
}

func runLoad(client httpDoer, targetURL string, totalRequests int64, concurrency int32) (*entity.Report, error) {
	started := time.Now()
	workers := int64(concurrency)
	if workers > totalRequests {
		workers = totalRequests
	}

	jobs := make(chan struct{})
	results := make(chan requestResult, workers)
	var workerGroup sync.WaitGroup
	workerGroup.Add(int(workers))
	for i := int64(0); i < workers; i++ {
		go func() {
			defer workerGroup.Done()
			for range jobs {
				results <- doRequest(client, targetURL)
			}
		}()
	}

	go func() {
		for i := int64(0); i < totalRequests; i++ {
			jobs <- struct{}{}
		}
		close(jobs)
	}()
	go func() {
		workerGroup.Wait()
		close(results)
	}()

	report := &entity.Report{
		TotalRequests: totalRequests,
		OrderStatus:   make(map[string]int),
	}
	for result := range results {
		if result.success {
			report.SuccessRequests++
			continue
		}

		status := "error"
		if result.status != 0 {
			status = strconv.Itoa(result.status)
		}
		report.OrderStatus[status]++
	}
	report.FinalTime = time.Since(started).Seconds()
	return report, nil
}

func doRequest(client httpDoer, targetURL string) requestResult {
	request, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return requestResult{}
	}

	response, err := client.Do(request)
	if err != nil || response == nil {
		return requestResult{}
	}
	if response.Body != nil {
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
	}

	return requestResult{
		status:  response.StatusCode,
		success: response.StatusCode == http.StatusOK,
	}
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var URL string
var TotalRequests int64
var Concurrency int32

func init() {
	rootCmd.Flags().StringVarP(&URL, "url", "u", "", "URL do servidor a ser testado")
	rootCmd.Flags().Int64VarP(&TotalRequests, "requests", "r", 1, "Número total de requisições a serem realizadas")
	rootCmd.Flags().Int32VarP(&Concurrency, "concurrency", "c", 1, "Número de chamadas simultâneas")
}
