package batch

import (
	"atoma/pkg/ingest"
	"atoma/pkg/tokenizer"
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Result struct {
	TotalTokens map[string]int
	TotalCost   map[string]float64
	FileCount   int
}

type Processor struct {
	Registry *tokenizer.Registry
	Models   []string
	Workers  int
}

func NewProcessor(registry *tokenizer.Registry, models []string, workers int) *Processor {
	if workers <= 0 {
		workers = 4
	}
	return &Processor{
		Registry: registry,
		Models:   models,
		Workers:  workers,
	}
}

func (p *Processor) ProcessDirectory(dirPath string) (*Result, error) {
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Base(path)[0] == '.' {
			return nil
		}
		files = append(files, path)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return p.ProcessFiles(files)
}

func (p *Processor) ProcessFiles(files []string) (*Result, error) {
	if len(files) == 0 {
		return &Result{
			TotalTokens: make(map[string]int),
			TotalCost:   make(map[string]float64),
			FileCount:   0,
		}, nil
	}

	fileChan := make(chan string, len(files))
	for _, f := range files {
		fileChan <- f
	}
	close(fileChan)

	var wg sync.WaitGroup
	var mu sync.Mutex
	
	totalTokens := make(map[string]int)
	totalCost := make(map[string]float64)
	fileCount := 0

	for i := 0; i < p.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileChan {
				if strings.HasSuffix(path, ".jsonl") {
					p.processJSONL(path, &mu, totalTokens, totalCost, &fileCount)
					continue
				}

				text, err := ingest.ExtractText(path)
				if err != nil {
					continue
				}
				data := []byte(text)

				messages, tools, isChat := p.detectSchema(data)

				mu.Lock()
				fileCount++
				for _, modelName := range p.Models {
					tk, err := p.Registry.Get(modelName)
					if err != nil {
						continue
					}

					var count int
					if isChat {
						count, _ = tk.CountMessages(messages)
					} else if len(tools) > 0 {
						count, _ = tk.CountTools(tools)
					} else {
						count, _ = tk.CountTokens(text)
					}
					
					info := tk.GetInfo()
					totalTokens[modelName] += count
					totalCost[modelName] += (float64(count) / 1000000.0) * info.InputPrice
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	return &Result{
		TotalTokens: totalTokens,
		TotalCost:   totalCost,
		FileCount:   fileCount,
	}, nil
}

func (p *Processor) detectSchema(data []byte) ([]tokenizer.Message, []tokenizer.Tool, bool) {
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

func (p *Processor) processJSONL(path string, mu *sync.Mutex, totalTokens map[string]int, totalCost map[string]float64, fileCount *int) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		messages, tools, isChat := p.detectSchema(line)

		mu.Lock()
		for _, modelName := range p.Models {
			tk, err := p.Registry.Get(modelName)
			if err != nil {
				continue
			}
			
			var count int
			if isChat {
				count, _ = tk.CountMessages(messages)
			} else if len(tools) > 0 {
				count, _ = tk.CountTools(tools)
			} else {
				count, _ = tk.CountTokens(string(line))
			}
			
			info := tk.GetInfo()
			totalTokens[modelName] += count
			totalCost[modelName] += (float64(count) / 1000000.0) * info.InputPrice
		}
		mu.Unlock()
	}
	mu.Lock()
	*fileCount++
	mu.Unlock()
}
