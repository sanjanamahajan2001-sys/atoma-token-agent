package ui

import (
	"atoma/pkg/optimizer"
	"atoma/pkg/tokenizer"
	"fmt"
	"sort"
	"strings"

	"github.com/pterm/pterm"
)

type turnInfo struct {
	Index  int
	Tokens int
	Role   string
}

type UI struct {
	configPath string
}

func NewUI() *UI {
	return &UI{}
}

func (u *UI) SetConfigPath(path string) {
	u.configPath = path
}

func (u *UI) DisplayAnalysis(results map[string]int, models map[string]tokenizer.ModelInfo, rawText string, isChat bool, actual int) {
	pterm.DefaultHeader.WithFullWidth().Println("Atoma: Token Analyser")
	if u.configPath != "" {
		pterm.FgGray.Printf("Using prices from: %s\n\n", u.configPath)
	}

	header := []string{"Provider", "Model", "Token Count", "Context Window", "Usage %", "Est. Cost ($)"}
	if actual > 0 {
		header = append(header, "Reasoning Delta")
	}
	tableData := pterm.TableData{header}

	for modelName, count := range results {
		info := models[modelName]
		usagePercent := (float64(count) / float64(info.ContextWindow)) * 100
		estCost := (float64(count) / 1000000.0) * info.InputPrice

		usageStr := fmt.Sprintf("%.2f%%", usagePercent)
		costStr := fmt.Sprintf("$%.4f", estCost)

		countStr := fmt.Sprintf("%d", count)
		if info.IsEstimated {
			countStr += pterm.FgGray.Sprint(" (est.)")
		}

		if count > info.ContextWindow {
			countStr = pterm.NewStyle(pterm.FgRed, pterm.Bold).Sprintf("%s [OVERFLOW]", countStr)
			usageStr = pterm.NewStyle(pterm.FgRed, pterm.Bold, pterm.Blink).Sprintf("%s [CRITICAL]", usageStr)
		} else if usagePercent > 80 {
			countStr = pterm.FgRed.Sprint(countStr)
			usageStr = pterm.FgRed.Sprint(usageStr)
		} else if usagePercent > 50 {
			countStr = pterm.FgYellow.Sprint(countStr)
			usageStr = pterm.FgYellow.Sprint(usageStr)
		}

		row := []string{
			fmt.Sprintf("%v", info.Provider),
			info.Name,
			countStr,
			fmt.Sprintf("%d", info.ContextWindow),
			usageStr,
			costStr,
		}

		if actual > 0 {
			delta := actual - count
			deltaStr := fmt.Sprintf("%d", delta)
			if delta > 0 {
				deltaStr = pterm.FgYellow.Sprintf("+%d (Reasoning)", delta)
			} else if delta < 0 {
				deltaStr = pterm.FgGreen.Sprintf("%d (Efficient)", delta)
			}
			row = append(row, deltaStr)
		}

		tableData = append(tableData, row)
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()

	if !isChat {
		pterm.DefaultSection.Println("Analysis Summary")
		pterm.Info.Printf("Input Length: %d characters\n", len(rawText))
		pterm.Info.Printf("Approximate Words: %d\n", len(strings.Fields(rawText)))
	}
}

func (u *UI) DisplayChatAnalysis(messages []tokenizer.Message, results map[string]int, models map[string]tokenizer.ModelInfo, tools []tokenizer.Tool, actual int) {
	pterm.DefaultHeader.WithFullWidth().Println("Atoma: Conversation Audit")
	if u.configPath != "" {
		pterm.FgGray.Printf("Using prices from: %s\n\n", u.configPath)
	}

	pterm.DefaultSection.Println("Turn-by-Turn Breakdown (Heatmap)")
	auditData := pterm.TableData{
		{"Turn", "Role", "Snippet", "Approx. Tokens"},
	}

	var history []turnInfo

	for i, msg := range messages {
		contentStr := fmt.Sprint(msg.Content)
		snippet := contentStr
		if len(snippet) > 50 {
			snippet = snippet[:47] + "..."
		}
		
		if len(msg.ToolCalls) > 0 {
			snippet = pterm.FgYellow.Sprintf("Tool Call: %s", msg.ToolCalls[0].Function.Name)
		}
		
		approxTokens := int(float64(len(strings.Fields(contentStr))) * 1.3)
		history = append(history, turnInfo{Index: i + 1, Tokens: approxTokens, Role: msg.Role})
		
		tokensStr := fmt.Sprintf("%d", approxTokens)
		if approxTokens > 2000 {
			tokensStr = pterm.NewStyle(pterm.FgRed, pterm.Bold).Sprint(tokensStr)
		} else if approxTokens > 500 {
			tokensStr = pterm.FgYellow.Sprint(tokensStr)
		} else {
			tokensStr = pterm.FgGreen.Sprint(tokensStr)
		}
		
		auditData = append(auditData, []string{
			fmt.Sprintf("%d", i+1),
			strings.Title(msg.Role),
			snippet,
			tokensStr,
		})
	}
	pterm.DefaultTable.WithHasHeader().WithData(auditData).Render()

	pterm.Println()
	u.DisplayAnalysis(results, models, "", true, actual)
	
	// Smart Pruning Recommendation
	u.suggestPruning(history)

	pterm.Println()
	pterm.DefaultSection.Println("Conversation Insights")
	pterm.Info.Printf("Total Turns: %d\n", len(messages))
	if len(tools) > 0 {
		pterm.Info.Printf("Tool Definitions: %d functions provided\n", len(tools))
	}
	pterm.Info.Println("Note: High-cost turns (>2,000 tokens) are highlighted in Red/Bold for pruning optimization.")
}

func (u *UI) suggestPruning(history []turnInfo) {
	if len(history) < 3 {
		return
	}

	// Sort by tokens descending
	sort.Slice(history, func(i, j int) bool {
		return history[i].Tokens > history[j].Tokens
	})

	var candidates []turnInfo
	for _, t := range history {
		if t.Tokens > 25 && t.Role != "system" {
			candidates = append(candidates, t)
		}
		if len(candidates) >= 3 {
			break
		}
	}

	if len(candidates) > 0 {
		pterm.Println()
		pterm.DefaultSection.Println("Agentic Pruning Recommendations")
		pterm.Info.Println("The following high-density turns are candidates for summarization or removal to reduce cost:")
		
		tableData := pterm.TableData{{"Turn #", "Role", "Tokens", "Impact"}}
		for _, c := range candidates {
			tableData = append(tableData, []string{
				fmt.Sprintf("%d", c.Index),
				strings.Title(c.Role),
				pterm.FgRed.Sprintf("%d", c.Tokens),
				"High Density",
			})
		}
		pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	}
}

func (u *UI) DisplayBatchAnalysis(totalTokens map[string]int, totalCost map[string]float64, models map[string]tokenizer.ModelInfo, fileCount int) {
	pterm.DefaultHeader.WithFullWidth().Println("Atoma: Batch Power Analysis")

	pterm.DefaultSection.Println("Batch Summary")
	pterm.Info.Printf("Total Files Processed: %d\n", fileCount)
	pterm.Println()

	tableData := pterm.TableData{
		{"Provider", "Model", "Total Tokens", "Total Est. Cost ($)", "Avg Tokens/File"},
	}

	for modelName, count := range totalTokens {
		info := models[modelName]
		cost := totalCost[modelName]
		avg := 0.0
		if fileCount > 0 {
			avg = float64(count) / float64(fileCount)
		}

		countStr := fmt.Sprintf("%d", count)
		if info.IsEstimated {
			countStr += pterm.FgGray.Sprint(" (est.)")
		}

		tableData = append(tableData, []string{
			fmt.Sprintf("%v", info.Provider),
			info.Name,
			countStr,
			fmt.Sprintf("$%.4f", cost),
			fmt.Sprintf("%.1f", avg),
		})
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

func (u *UI) ShowError(err error) { pterm.Error.Println(err.Error()) }
func (u *UI) ShowSuccess(msg string) { pterm.Success.Println(msg) }

func (u *UI) DisplayOptimization(res optimizer.Result) {
	pterm.DefaultHeader.WithFullWidth().Println("Atoma: Prompt Optimizer")
	pterm.DefaultSection.Println("Optimization Suggestions")
	for _, sugg := range res.Suggestions {
		pterm.Info.Printf("Remove '%s' (Reason: %s)\n", pterm.FgRed.Sprint(sugg.Original), sugg.Reason)
	}
	pterm.Println()
	pterm.DefaultSection.Println("Side-by-Side Comparison")
	panels := pterm.Panels{{{Data: pterm.FgCyan.Sprint("Original") + "\n\n" + res.OriginalText}}, {{Data: pterm.FgGreen.Sprint("Optimized") + "\n\n" + res.OptimizedText}}}
	_ = pterm.DefaultPanel.WithPanels(panels).Render()
	pterm.Println()
	pterm.DefaultSection.Println("Efficiency Gains")
	tableData := pterm.TableData{{"Metric", "Original", "Optimized", "Savings"}, {"Tokens", fmt.Sprintf("%d", res.TokensOriginal), fmt.Sprintf("%d", res.TokensOriginal-res.TokensSaved), pterm.FgGreen.Sprintf("%d (%d%%)", res.TokensSaved, int(res.PercentSaved))}}
	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}
