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

func (c *Client) completion(ctx context.Context, prompt string) (gpt.CompletionResponse, error) {
	var completionResponse gpt.CompletionResponse

	fn := func() error {
		completionRequest := newCompletionRequest(prompt)

		var err error
		completionResponse, err = c.gpt3Client.CreateCompletion(ctx, completionRequest)
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

func newCompletionRequest(prompt string) gpt.CompletionRequest {
	var (
		maxTokens           = 512
		temperature float32 = 0.0
		stop                = []string{"'''"}
	)

	completionRequest := gpt.CompletionRequest{
		Model:       gpt.GPT3TextDavinci003,
		Prompt:      prompt,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Stop:        stop,
	}

	return completionRequest
}
