package main

import (
	"atoma/pkg/batch"
	"atoma/pkg/export"
	"atoma/pkg/ingest"
	"atoma/pkg/optimizer"
	"atoma/pkg/tokenizer"
	"atoma/pkg/ui"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	modelName    string
	filePath     string
	compare      bool
	chatMode     bool
	dirPath      string
	workers      int
	exportPath   string
	configPath   string
	actualTokens int
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "atoma",
		Short: "Atoma is a high-precision Token Analyser Agent",
		Long:  `A production-ready CLI for estimating token usage and cost across major LLM providers.`,
	}

	var analyzeCmd = &cobra.Command{
		Use:   "analyze [text]",
		Short: "Analyze token count for a given text or file",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appUI := ui.NewUI()
			appUI.SetConfigPath(configPath)
			var text string
			var data []byte
			var err error
			
			if filePath != "" {
				text, err = ingest.ExtractText(filePath)
				if err != nil {
					appUI.ShowError(fmt.Errorf("failed to ingest file: %w", err))
					return
				}
				data = []byte(text)
			} else if len(args) > 0 {
				text = args[0]
				data = []byte(text)
			} else {
				appUI.ShowError(fmt.Errorf("please provide text or a file path using -f"))
				return
			}

			messages, tools, isChat := detectSchema(data, chatMode)
			registry := tokenizer.NewRegistry()
			setupRegistry(registry, appUI, configPath)

			results := make(map[string]int)
			modelInfos := make(map[string]tokenizer.ModelInfo)
			modelList := getModelList(compare, modelName)

			for _, m := range modelList {
				t, err := registry.Get(m)
				if err != nil {
					appUI.ShowError(fmt.Errorf("model '%s' not found in registry (check configs/prices.yaml)", m))
					continue
				}
				
				var count int
				if isChat {
					count, _ = t.CountMessages(messages)
				} else if len(tools) > 0 {
					count, _ = t.CountTools(tools)
				} else {
					count, _ = t.CountTokens(text)
				}
				
				results[m] = count
				modelInfos[m] = t.GetInfo()
			}

			if len(results) == 0 {
				appUI.ShowError(fmt.Errorf("no valid models found to perform analysis. Use --compare to view standard providers."))
				return
			}

			// Rendering
			if isChat {
				appUI.DisplayChatAnalysis(messages, results, modelInfos, tools, actualTokens)
			} else {
				appUI.DisplayAnalysis(results, modelInfos, text, false, actualTokens)
			}

			// Exporting
			if exportPath != "" {
				err := export.SaveAnalysisToCSV(exportPath, results, modelInfos)
				if err != nil {
					appUI.ShowError(fmt.Errorf("failed to export CSV: %w", err))
				} else {
					appUI.ShowSuccess(fmt.Sprintf("Report exported to %s", exportPath))
				}
			}
		},
	}

	var batchCmd = &cobra.Command{
		Use:   "batch",
		Short: "Process multiple files or directories for high-volume analysis",
		Run: func(cmd *cobra.Command, args []string) {
			appUI := ui.NewUI()
			appUI.SetConfigPath(configPath)
			registry := tokenizer.NewRegistry()
			setupRegistry(registry, appUI, configPath)

			modelList := getModelList(compare, modelName)
			processor := batch.NewProcessor(registry, modelList, workers)

			var res *batch.Result
			var err error

			if dirPath != "" {
				res, err = processor.ProcessDirectory(dirPath)
			} else if filePath != "" {
				res, err = processor.ProcessFiles([]string{filePath})
			} else {
				appUI.ShowError(fmt.Errorf("please provide --dir or --file path"))
				return
			}
			if err != nil {
				appUI.ShowError(err)
				return
			}

			modelInfos := make(map[string]tokenizer.ModelInfo)
			for _, m := range modelList {
				t, _ := registry.Get(m)
				modelInfos[m] = t.GetInfo()
			}

			appUI.DisplayBatchAnalysis(res.TotalTokens, res.TotalCost, modelInfos, res.FileCount)

			if exportPath != "" {
				err := export.SaveBatchToCSV(exportPath, res.TotalTokens, res.TotalCost, modelInfos, res.FileCount)
				if err != nil {
					appUI.ShowError(fmt.Errorf("failed to export batch CSV: %w", err))
				} else {
					appUI.ShowSuccess(fmt.Sprintf("Batch report exported to %s", exportPath))
				}
			}
		},
	}

	var optimizeCmd = &cobra.Command{
		Use:   "optimize [text]",
		Short: "Suggest optimizations to reduce token count",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			appUI := ui.NewUI()
			appUI.SetConfigPath(configPath)
			var text string
			if filePath != "" {
				var err error
				text, err = ingest.ExtractText(filePath)
				if err != nil {
					appUI.ShowError(fmt.Errorf("failed to ingest file: %w", err))
					return
				}
			} else if len(args) > 0 {
				text = args[0]
			} else {
				appUI.ShowError(fmt.Errorf("please provide text or a file path using -f"))
				return
			}

			opt := optimizer.NewTextOptimizer()
			optimized, suggestions := opt.Run(text)
			tk, _ := tokenizer.NewOpenAITokenizer("gpt-4o")
			origTokens, _ := tk.CountTokens(text)
			optTokens, _ := tk.CountTokens(optimized)
			saved := origTokens - optTokens
			percent := 0.0
			if origTokens > 0 {
				percent = (float64(saved) / float64(origTokens)) * 100
			}
			appUI.DisplayOptimization(optimizer.Result{
				OriginalText: text, OptimizedText: optimized, Suggestions: suggestions,
				TokensOriginal: origTokens, TokensSaved: saved, PercentSaved: percent,
			})
		},
	}

	analyzeFlags(analyzeCmd)
	batchFlags(batchCmd)
	optimizeCmd.Flags().StringVarP(&filePath, "file", "f", "", "File path to read text from")

	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(batchCmd)
	rootCmd.AddCommand(optimizeCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func detectSchema(data []byte, chatMode bool) ([]tokenizer.Message, []tokenizer.Tool, bool) {
	if !tokenizer.IsJSON(data) {
		return nil, nil, false
	}
	msgs, err := tokenizer.ParseChat(data)
	if err == nil {
		return msgs, nil, true
	}
	var toolPayload struct {
		Tools []tokenizer.Tool `json:"tools"`
	}
	if err := json.Unmarshal(data, &toolPayload); err == nil && len(toolPayload.Tools) > 0 {
		return nil, toolPayload.Tools, false
	}
	return nil, nil, false
}

func analyzeFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&modelName, "model", "m", "gpt-4o", "Model to use for analysis")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "File path to read text from")
	cmd.Flags().BoolVarP(&compare, "compare", "c", false, "Compare counts across major providers")
	cmd.Flags().BoolVarP(&chatMode, "chat", "t", false, "Enable chat mode")
	cmd.Flags().StringVarP(&exportPath, "export", "e", "", "CSV file to export results to")
	cmd.Flags().StringVarP(&configPath, "config", "k", "configs/prices.yaml", "Path to prices.yaml")
	cmd.Flags().IntVarP(&actualTokens, "actual", "a", 0, "Actual billed tokens from provider for Reasoning Delta audit")
}

func batchFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&dirPath, "dir", "d", "", "Directory path to analyze")
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Specific file path to analyze")
	cmd.Flags().StringVarP(&modelName, "model", "m", "gpt-4o", "Model for analysis")
	cmd.Flags().BoolVarP(&compare, "compare", "c", false, "Compare providers")
	cmd.Flags().IntVarP(&workers, "workers", "w", 4, "Number of parallel workers")
	cmd.Flags().StringVarP(&exportPath, "export", "e", "", "CSV file to export results to")
	cmd.Flags().StringVarP(&configPath, "config", "k", "configs/prices.yaml", "Path to prices.yaml")
}

func getModelList(compare bool, modelName string) []string {
	if compare {
		return []string{"gpt-4o", "claude-3.5", "gemini-1.5-pro"}
	}
	return []string{modelName}
}

func setupRegistry(registry *tokenizer.Registry, appUI *ui.UI, configPath string) {
	// Initialize defaults
	tkO, _ := tokenizer.NewOpenAITokenizer("gpt-4o")
	registry.Register("gpt-4o", tkO)
	registry.Register("claude-3.5", tokenizer.NewAnthropicTokenizer("claude-3.5"))
	registry.Register("gemini-1.5-pro", tokenizer.NewGeminiTokenizer("gemini-1.5-pro"))

	// Load dynamic config if exists
	config, err := tokenizer.LoadConfig(configPath)
	if err == nil {
		for _, mI := range config.Models {
			var tk tokenizer.Tokenizer
			var err error
			if mI.Provider == tokenizer.OpenAI {
				tk, err = tokenizer.NewOpenAITokenizer(mI.Name)
			} else if mI.Provider == tokenizer.Anthropic {
				tk = tokenizer.NewAnthropicTokenizer(mI.Name)
			} else {
				tk = tokenizer.NewGeminiTokenizer(mI.Name)
			}
			if err == nil {
				tk.UpdateInfo(mI)
				registry.Register(mI.Name, tk)
			}
		}
	} else if configPath != "configs/prices.yaml" {
		appUI.ShowError(fmt.Errorf("failed to load config %s: %v", configPath, err))
	}
}
