package token

import (
	"hash/fnv"
	"runtime"
	"strings"
	"sync"

	"github.com/sokinpui/coder/internal/types"
	"github.com/tiktoken-go/tokenizer"
)

var (
	enc        tokenizer.Codec
	encOnce    sync.Once
	tokenCache sync.Map
)

type cacheKey struct {
	length int
	hash   uint64
}

func getEncoder() tokenizer.Codec {
	encOnce.Do(func() {
		var err error
		enc, err = tokenizer.Get(tokenizer.Cl100kBase)
		if err != nil {
			enc, _ = tokenizer.Get(tokenizer.O200kBase)
		}
	})
	return enc
}

func CountTokens(messages []types.Message) int {
	if len(messages) == 0 {
		return 0
	}

	encoder := getEncoder()
	if len(messages) == 1 {
		return countSingleMessage(messages[0], encoder)
	}

	counts := make([]int, len(messages))
	var uncachedIndices []int

	for i, msg := range messages {
		if !msg.CanSendToAI() {
			continue
		}
		if msg.Type == types.ImageMessage {
			counts[i] = 1500
			continue
		}
		if msg.Content == "" {
			continue
		}

		key := getCacheKey(msg.Content)
		if val, ok := tokenCache.Load(key); ok {
			counts[i] = val.(int)
			continue
		}
		uncachedIndices = append(uncachedIndices, i)
	}

	if len(uncachedIndices) == 0 {
		return sumCounts(counts)
	}

	if len(uncachedIndices) == 1 {
		idx := uncachedIndices[0]
		counts[idx] = countAndCache(messages[idx].Content, encoder)
		return sumCounts(counts)
	}

	workerCount := min(len(uncachedIndices), runtime.NumCPU())
	jobs := make(chan int, len(uncachedIndices))
	for _, idx := range uncachedIndices {
		jobs <- idx
	}
	close(jobs)

	var wg sync.WaitGroup
	for range workerCount {
		wg.Go(func() {
			for idx := range jobs {
				counts[idx] = countAndCache(messages[idx].Content, encoder)
			}
		})
	}
	wg.Wait()

	return sumCounts(counts)
}

func countSingleMessage(msg types.Message, encoder tokenizer.Codec) int {
	if !msg.CanSendToAI() {
		return 0
	}
	if msg.Type == types.ImageMessage {
		return 1500
	}
	if msg.Content == "" {
		return 0
	}
	return countAndCache(msg.Content, encoder)
}

func countAndCache(content string, encoder tokenizer.Codec) int {
	key := getCacheKey(content)
	if val, ok := tokenCache.Load(key); ok {
		return val.(int)
	}

	count := encodeTokens(content, encoder)
	tokenCache.Store(key, count)
	return count
}

func sumCounts(counts []int) int {
	total := 0
	for _, c := range counts {
		total += c
	}
	return total
}

func getCacheKey(s string) cacheKey {
	h := fnv.New64a()
	h.Write([]byte(s))
	return cacheKey{
		length: len(s),
		hash:   h.Sum64(),
	}
}

func encodeTokens(content string, encoder tokenizer.Codec) int {
	if encoder != nil {
		if len(content) > 64*1024 && runtime.NumCPU() > 1 {
			return encodeTokensParallel(content, encoder)
		}
		ids, _, err := encoder.Encode(content)
		if err == nil {
			return len(ids)
		}
	}
	return estimateTokensFallback(content)
}

func encodeTokensParallel(content string, encoder tokenizer.Codec) int {
	lines := strings.Split(content, "\n")
	if len(lines) <= 1 {
		ids, _, err := encoder.Encode(content)
		if err == nil {
			return len(ids)
		}
		return estimateTokensFallback(content)
	}

	workerCount := min(runtime.NumCPU(), len(lines))
	chunkSize := (len(lines) + workerCount - 1) / workerCount
	counts := make([]int, workerCount)
	var wg sync.WaitGroup

	for w := range workerCount {
		start := w * chunkSize
		if start >= len(lines) {
			break
		}
		end := min(start+chunkSize, len(lines))

		wg.Add(1)
		go func(idx, s, e int) {
			defer wg.Done()
			chunkText := strings.Join(lines[s:e], "\n")
			if e < len(lines) {
				chunkText += "\n"
			}
			ids, _, err := encoder.Encode(chunkText)
			if err == nil {
				counts[idx] = len(ids)
				return
			}
			counts[idx] = estimateTokensFallback(chunkText)
		}(w, start, end)
	}
	wg.Wait()
	return sumCounts(counts)
}

func estimateTokensFallback(text string) int {
	// Simple fallback: ~4 characters per token average
	return len(text) / 4
}
