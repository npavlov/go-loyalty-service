package orders

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/npavlov/go-loyalty-service/internal/config"
	"github.com/npavlov/go-loyalty-service/internal/models"
)

var CantProcessError = errors.New("can't process orders")

type Sender struct {
	cfg *config.Config
	l   *zerolog.Logger
}

func NewSender(cfg *config.Config, logger *zerolog.Logger) *Sender {
	return &Sender{
		cfg: cfg,
		l:   logger,
	}
}

func (sender *Sender) SendPostRequest(ctx context.Context, orderNumber string) (*models.Accrual, error) {
	// Создаем HTTP-клиент
	client := resty.New()

	// Формируем URL для внешнего сервиса
	url := fmt.Sprintf("%s/api/orders/%s", sender.cfg.AccrualAddress, orderNumber)

	// Выполняем запрос
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		Get(url)
	if err != nil {
		sender.l.Error().Err(err).Send()

		return nil, errors.Wrap(err, "could not send request")
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		var response models.Accrual
		if err := json.Unmarshal(resp.Body(), &response); err != nil {
			sender.l.Error().Err(err).Msg("failed to unmarshal response")

			return nil, errors.Wrap(err, "failed to unmarshal response")
		}

		return &response, nil
	case http.StatusNoContent:
	case http.StatusTooManyRequests:
	case http.StatusInternalServerError:
	default:
		sender.l.Error().Int("status", resp.StatusCode()).Msg("Can't process orders, retry")
	}

	return nil, CantProcessError
}
