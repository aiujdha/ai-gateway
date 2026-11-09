package relay

import (
	"context"
	"fmt"
	"unicode/utf8"
)

// LocalProvider returns deterministic text, without contacting a model service.
type LocalProvider struct{ Channel string }

func (p LocalProvider) Name() string { return p.Channel }
func (p LocalProvider) Generate(ctx context.Context, request Request) (Response, error) {
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}
	text := fmt.Sprintf("[%s] 已收到：%s。这是本地适配器返回的文本，可用于检查统一接口、流式响应和渠道切换。", p.Channel, request.Prompt)
	return Response{Text: text, InputTokens: utf8.RuneCountInString(request.Prompt), OutputTokens: utf8.RuneCountInString(text)}, nil
}
func (p LocalProvider) Stream(ctx context.Context, request Request) (<-chan StreamEvent, error) {
	response, err := p.Generate(ctx, request)
	if err != nil {
		return nil, err
	}
	events := make(chan StreamEvent)
	go func() {
		defer close(events)
		for _, character := range response.Text {
			select {
			case events <- StreamEvent{Delta: string(character)}:
			case <-ctx.Done():
				return
			}
		}
		select {
		case events <- StreamEvent{Done: true}:
		case <-ctx.Done():
		}
	}()
	return events, nil
}
