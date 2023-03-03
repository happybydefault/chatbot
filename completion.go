package chatbot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	gpt "github.com/sashabaranov/go-gpt3"
	"go.uber.org/zap"
)

func (c *Client) completion(ctx context.Context, messages []gpt.ChatCompletionMessage) (gpt.ChatCompletionResponse, error) {
	var completionResponse gpt.ChatCompletionResponse

	fn := func() error {
		completionRequest := newCompletionRequest(messages)

		var err error
		completionResponse, err = c.gpt3Client.CreateChatCompletion(ctx, completionRequest)
		if err != nil {
			var apiErr *gpt.APIError
			if errors.As(err, &apiErr) {
				if apiErr.StatusCode < 500 || apiErr.StatusCode >= 600 {
					return backoff.Permanent(err)
				}
			}
			c.logger.Debug("failed attempt to get completion response", zap.Error(err))
			return err
		}

		if len(completionResponse.Choices) == 0 {
			return backoff.Permanent(errors.New("received empty slice of completion choices"))
		}

		c.logger.Debug(
			"received completion response",
			zap.String("completion_response", fmt.Sprintf("%#v", completionResponse)),
		)

		return nil
	}

	err := backoff.Retry(
		fn,
		backoff.WithMaxRetries(backoff.NewConstantBackOff(100*time.Millisecond), 3),
	)

	return completionResponse, err
}

func newCompletionRequest(messages []gpt.ChatCompletionMessage) gpt.ChatCompletionRequest {
	var (
		maxTokens           = 512
		temperature float32 = 0.0
		stop                = []string{"'''"}
	)

	completionRequest := gpt.ChatCompletionRequest{
		Model:       gpt.GPT3Dot5Turbo,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Stop:        stop,
	}

	return completionRequest
}
