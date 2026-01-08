package tokenizer

import (
	"fmt"
	"sync"

	"github.com/pkoukk/tiktoken-go"
)

var (
	defaultEncoding = "cl100k_base"
	cache           sync.Map // map[string]*tiktoken.Tiktoken
)

// GetEncoding 获取指定模型的编码器
func GetEncoding(model string) (*tiktoken.Tiktoken, error) {
	// 检查缓存
	if v, ok := cache.Load(model); ok {
		return v.(*tiktoken.Tiktoken), nil
	}

	// 尝试根据模型名获取
	tkm, err := tiktoken.EncodingForModel(model)
	if err != nil {
		// 如果找不到模型，回退到 cl100k_base
		tkm, err = tiktoken.GetEncoding(defaultEncoding)
		if err != nil {
			return nil, fmt.Errorf("failed to get encoding: %w", err)
		}
	}

	cache.Store(model, tkm)
	return tkm, nil
}

// CountTokens 计算文本的 Token 数
func CountTokens(model, text string) int {
	tkm, err := GetEncoding(model)
	if err != nil {
		// 降级策略：回退到估算
		return len(text) / 4
	}
	return len(tkm.Encode(text, nil, nil))
}
