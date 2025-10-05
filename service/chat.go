package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/atopos31/llmio/balancer"
	"github.com/atopos31/llmio/middleware"
	"github.com/atopos31/llmio/models"
	"github.com/atopos31/llmio/providers"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func BalanceChat(c *gin.Context, style string, Beforer Beforer, processer Processer) error {
	requestID := middleware.GetRequestID(c)
	proxyStart := time.Now()

	slog.Info("balance_chat_started",
		"request_id", requestID,
		"style", style,
	)

	rawData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		slog.Error("failed_to_read_request_body",
			"request_id", requestID,
			"error", err,
		)
		return err
	}

	ctx := c.Request.Context()
	before, err := Beforer(rawData)
	if err != nil {
		slog.Error("failed_to_parse_request",
			"request_id", requestID,
			"error", err,
		)
		return err
	}

	slog.Info("request_parsed",
		"request_id", requestID,
		"model", before.model,
		"stream", before.stream,
		"tool_call", before.toolCall,
		"structured_output", before.structuredOutput,
		"image", before.image,
	)

	llmProvidersWithLimit, err := ProvidersBymodelsName(ctx, before.model)
	if err != nil {
		slog.Error("failed_to_get_providers",
			"request_id", requestID,
			"model", before.model,
			"error", err,
		)
		return err
	}
	// 所有模型提供商关联
	llmproviders := llmProvidersWithLimit.Providers

	slog.Info("providers_found",
		"request_id", requestID,
		"model", before.model,
		"provider_count", len(llmproviders),
		"max_retry", llmProvidersWithLimit.MaxRetry,
		"timeout", llmProvidersWithLimit.TimeOut,
	)

	if len(llmproviders) == 0 {
		return fmt.Errorf("no provider found for models %s", before.model)
	}

	// 预分配切片容量
	providerIds := make([]uint, 0, len(llmproviders))
	for _, modelWithProvider := range llmproviders {
		providerIds = append(providerIds, modelWithProvider.ProviderID)
	}

	provideritems, err := gorm.G[models.Provider](models.DB).Where("id IN ?", providerIds).Where("type = ?", style).Find(ctx)
	if err != nil {
		return err
	}
	if len(provideritems) == 0 {
		return fmt.Errorf("no %s provider found for %s", style, before.model)
	}

	// 构建providerID到provider的映射，避免重复查找
	providerMap := make(map[uint]*models.Provider, len(provideritems))
	for i := range provideritems {
		provider := &provideritems[i]
		providerMap[provider.ID] = provider
	}

	items := make(map[uint]int)
	for _, modelWithProvider := range llmproviders {
		// 过滤是否开启工具调用
		if modelWithProvider.ToolCall != nil && before.toolCall && !*modelWithProvider.ToolCall {
			continue
		}
		// 过滤是否开启结构化输出
		if modelWithProvider.StructuredOutput != nil && before.structuredOutput && !*modelWithProvider.StructuredOutput {
			continue
		}
		// 过滤是否拥有视觉能力
		if modelWithProvider.Image != nil && before.image && !*modelWithProvider.Image {
			continue
		}
		provider := providerMap[modelWithProvider.ProviderID]
		// 过滤提供商类型
		if provider == nil || provider.Type != style {
			continue
		}
		items[modelWithProvider.ID] = modelWithProvider.Weight
	}

	if len(items) == 0 {
		slog.Error("no_valid_provider_after_filtering",
			"request_id", requestID,
			"model", before.model,
			"tool_call", before.toolCall,
			"structured_output", before.structuredOutput,
			"image", before.image,
		)
		return errors.New("no provider with tool_call or structured_output or image found for models " + before.model)
	}

	slog.Info("load_balancing_ready",
		"request_id", requestID,
		"available_providers", len(items),
	)
	// 收集重试过程中的err日志
	retryErrLog := make(chan models.ChatLog, llmProvidersWithLimit.MaxRetry)
	defer close(retryErrLog)
	go func() {
		for log := range retryErrLog {
			_, err := SaveChatLog(context.Background(), log)
			if err != nil {
				slog.Error("save chat log error", "error", err)
			}
		}
	}()

	for retry := 0; retry < llmProvidersWithLimit.MaxRetry; retry++ {
		slog.Info("retry_attempt",
			"request_id", requestID,
			"retry", retry,
			"max_retry", llmProvidersWithLimit.MaxRetry,
		)

		select {
		case <-ctx.Done():
			slog.Warn("request_cancelled",
				"request_id", requestID,
				"retry", retry,
			)
			return ctx.Err()
		case <-time.After(time.Second * time.Duration(llmProvidersWithLimit.TimeOut)):
			slog.Error("retry_timeout",
				"request_id", requestID,
				"timeout_seconds", llmProvidersWithLimit.TimeOut,
			)
			return errors.New("retry time out !")
		default:
			// 加权负载均衡
			item, err := balancer.WeightedRandom(items)
			if err != nil {
				return err
			}
			modelWithProviderIndex := slices.IndexFunc(llmproviders, func(mp models.ModelWithProvider) bool {
				return mp.ID == *item
			})
			modelWithProvider := llmproviders[modelWithProviderIndex]

			provider := providerMap[modelWithProvider.ProviderID]

			chatModel, err := providers.New(style, provider.Config)
			if err != nil {
				slog.Error("failed_to_create_provider_client",
					"request_id", requestID,
					"provider", provider.Name,
					"error", err,
				)
				return err
			}

			slog.Info("provider_selected",
				"request_id", requestID,
				"provider", provider.Name,
				"provider_model", modelWithProvider.ProviderModel,
				"retry", retry,
			)

			log := models.ChatLog{
				Name:          before.model,
				ProviderModel: modelWithProvider.ProviderModel,
				ProviderName:  provider.Name,
				Status:        "success",
				Style:         style,
				Retry:         retry,
				ProxyTime:     time.Since(proxyStart),
			}
			reqStart := time.Now()
			client := providers.GetClient(time.Second * time.Duration(llmProvidersWithLimit.TimeOut) / 3)

			slog.Info("sending_request_to_provider",
				"request_id", requestID,
				"provider", provider.Name,
				"timeout_seconds", llmProvidersWithLimit.TimeOut/3,
			)

			res, err := chatModel.Chat(ctx, client, modelWithProvider.ProviderModel, before.raw)
			if err != nil {
				slog.Error("provider_request_failed",
					"request_id", requestID,
					"provider", provider.Name,
					"retry", retry,
					"error", err,
				)
				retryErrLog <- log.WithError(err)
				// 请求失败 移除待选
				delete(items, *item)
				continue
			}

			if res.StatusCode != http.StatusOK {
				byteBody, err := io.ReadAll(res.Body)
				if err != nil {
					slog.Error("read body error", "error", err)
				}
				slog.Error("provider_returned_error_status",
					"request_id", requestID,
					"provider", provider.Name,
					"status_code", res.StatusCode,
					"response_body", string(byteBody),
					"retry", retry,
				)
				retryErrLog <- log.WithError(fmt.Errorf("status: %d, body: %s", res.StatusCode, string(byteBody)))

				if res.StatusCode == http.StatusTooManyRequests {
					slog.Warn("rate_limit_hit",
						"request_id", requestID,
						"provider", provider.Name,
					)
					// 达到RPM限制 降低权重
					items[*item] -= items[*item] / 3
				} else {
					// 非RPM限制 移除待选
					delete(items, *item)
				}
				res.Body.Close()
				continue
			}
			defer res.Body.Close()

			slog.Info("provider_response_success",
				"request_id", requestID,
				"provider", provider.Name,
				"status_code", res.StatusCode,
			)

			logId, err := SaveChatLog(ctx, log)
			if err != nil {
				return err
			}

			pr, pw := io.Pipe()
			tee := io.TeeReader(res.Body, pw)

			// 与客户端并行处理响应数据流 同时记录日志
			go func(ctx context.Context) {
				defer pr.Close()
				processer(ctx, pr, before.stream, logId, reqStart)
			}(context.Background())
			// 转发给客户端
			if before.stream {
				c.Header("Content-Type", "text/event-stream")
				c.Header("Cache-Control", "no-cache")
			} else {
				c.Header("Content-Type", "application/json")
			}
			c.Writer.Flush()
			if _, err := io.Copy(c.Writer, tee); err != nil {
				pw.CloseWithError(err)
				return err
			}

			pw.Close()

			return nil
		}
	}

	return errors.New("maximum retry attempts reached !")
}

func SaveChatLog(ctx context.Context, log models.ChatLog) (uint, error) {
	if err := gorm.G[models.ChatLog](models.DB).Create(ctx, &log); err != nil {
		return 0, err
	}
	return log.ID, nil
}

type ProvidersWithlimit struct {
	Providers []models.ModelWithProvider
	MaxRetry  int
	TimeOut   int
}

func ProvidersBymodelsName(ctx context.Context, modelsName string) (*ProvidersWithlimit, error) {
	slog.Debug("query_model_providers_started",
		"model", modelsName,
	)

	// 明确排除软删除的记录
	llmmodels, err := gorm.G[models.Model](models.DB).Where("name = ? AND deleted_at IS NULL", modelsName).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Warn("model_not_found",
				"model", modelsName,
			)
			return nil, errors.New("not found model " + modelsName)
		}
		slog.Error("database_query_error",
			"model", modelsName,
			"error", err,
		)
		return nil, err
	}

	slog.Debug("model_found",
		"model", modelsName,
		"model_id", llmmodels.ID,
		"max_retry", llmmodels.MaxRetry,
		"timeout", llmmodels.TimeOut,
	)

	llmproviders, err := gorm.G[models.ModelWithProvider](models.DB).Where("model_id = ?", llmmodels.ID).Find(ctx)
	if err != nil {
		slog.Error("failed_to_query_model_providers",
			"model", modelsName,
			"model_id", llmmodels.ID,
			"error", err,
		)
		return nil, err
	}

	if len(llmproviders) == 0 {
		slog.Error("no_provider_mapping_found",
			"model", modelsName,
			"model_id", llmmodels.ID,
		)
		return nil, errors.New("not provider for model " + modelsName)
	}

	slog.Info("model_providers_loaded",
		"model", modelsName,
		"model_id", llmmodels.ID,
		"provider_count", len(llmproviders),
	)

	return &ProvidersWithlimit{
		Providers: llmproviders,
		MaxRetry:  llmmodels.MaxRetry,
		TimeOut:   llmmodels.TimeOut,
	}, nil
}
