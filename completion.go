package chatbot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PullRequestInc/go-gpt3"
	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
)

func (c *Client) completion(ctx context.Context, prompt string) (*gpt3.CompletionResponse, error) {
	var completionResponse *gpt3.CompletionResponse

	fn := func() error {
		completionRequest := newCompletionRequest([]string{prompt})

		var err error
		completionResponse, err = c.gpt3Client.Completion(ctx, completionRequest)
		if err != nil {
			var apiErr *gpt3.APIError
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

func newCompletionRequest(prompts []string) gpt3.CompletionRequest {
	var (
		maxTokens           = 512
		temperature float32 = 0.0
		stop                = []string{"'''"}
	)

	completionRequest := gpt3.CompletionRequest{
		Prompt:      prompts,
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
		Stop:        stop,
	}

	return completionRequest
}
