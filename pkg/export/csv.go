package export

import (
	"atoma/pkg/tokenizer"
	"encoding/csv"
	"fmt"
	"os"
)

// SaveAnalysisToCSV exports single analysis results to a CSV file
func SaveAnalysisToCSV(path string, results map[string]int, models map[string]tokenizer.ModelInfo) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Header
	writer.Write([]string{"Provider", "Model", "Token Count", "Context Window", "Input Price (1M)", "Est. Cost ($)"})

	for modelName, count := range results {
		info := models[modelName]
		cost := (float64(count) / 1000000.0) * info.InputPrice
		writer.Write([]string{
			fmt.Sprintf("%s", info.Provider),
			info.Name,
			fmt.Sprintf("%d", count),
			fmt.Sprintf("%d", info.ContextWindow),
			fmt.Sprintf("%.2f", info.InputPrice),
			fmt.Sprintf("%.4f", cost),
		})
	}

	return nil
}

// SaveBatchToCSV exports batch results to a CSV file
func SaveBatchToCSV(path string, totalTokens map[string]int, totalCost map[string]float64, models map[string]tokenizer.ModelInfo, fileCount int) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Header
	writer.Write([]string{"Provider", "Model", "Total Tokens", "Total Est. Cost ($)", "Avg Tokens/File", "File Count"})

	for modelName, count := range totalTokens {
		info := models[modelName]
		cost := totalCost[modelName]
		avg := 0.0
		if fileCount > 0 {
			avg = float64(count) / float64(fileCount)
		}

		writer.Write([]string{
			fmt.Sprintf("%s", info.Provider),
			info.Name,
			fmt.Sprintf("%d", count),
			fmt.Sprintf("%.4f", cost),
			fmt.Sprintf("%.2f", avg),
			fmt.Sprintf("%d", fileCount),
		})
	}

	return nil
}
