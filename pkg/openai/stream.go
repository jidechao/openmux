package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// StreamReader SSE 流读取器
type StreamReader struct {
	reader  *bufio.Reader
	ctx     context.Context
}

// NewStreamReader 创建流读取器
func NewStreamReader(ctx context.Context, r io.Reader) *StreamReader {
	return &StreamReader{
		reader: bufio.NewReader(r),
		ctx:    ctx,
	}
}

// Recv 接收下一个块
func (s *StreamReader) Recv() (*ChatCompletionChunk, error) {
	for {
		select {
		case <-s.ctx.Done():
			return nil, s.ctx.Err()
		default:
		}

		line, err := s.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return nil, io.EOF
		}

		var chunk ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, fmt.Errorf("failed to unmarshal chunk: %w", err)
		}

		// 确保 object 字段符合 OpenAI 标准
		if chunk.Object == "" {
			chunk.Object = "chat.completion.chunk"
		}

		return &chunk, nil
	}
}

// Close 关闭流
func (s *StreamReader) Close() error {
	return nil
}
