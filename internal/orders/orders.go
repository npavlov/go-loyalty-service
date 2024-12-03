package orders

import (
	"context"
	"encoding/json"
	"log"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"

	"github.com/npavlov/go-loyalty-service/internal/config"
	"github.com/npavlov/go-loyalty-service/internal/storage"
	"github.com/npavlov/go-loyalty-service/internal/utils"
)

type KafkaOrder struct {
	OrderNum string `json:"orderNum"`
	UserID   string `json:"userId"`
}

type Orders struct {
	log     *zerolog.Logger
	writer  *kafka.Writer
	reader  *kafka.Reader
	storage *storage.DBStorage
	sender  *Sender
}

func NewOrders(writer *kafka.Writer, reader *kafka.Reader, log *zerolog.Logger) *Orders {
	//nolint:exhaustruct
	return &Orders{
		log:    log,
		writer: writer,
		reader: reader,
	}
}

func (or *Orders) WithSender(cfg *config.Config) *Orders {
	or.sender = NewSender(cfg, or.log)

	return or
}

func (or *Orders) WithStorage(storage *storage.DBStorage) *Orders {
	or.storage = storage

	return or
}

func (or *Orders) AddOrder(ctx context.Context, orderNum string, userID string) error {
	data := KafkaOrder{
		OrderNum: orderNum,
		UserID:   userID,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		or.log.Error().Err(err).Msg("Error marshalling data for Kafka")

		return errors.Wrap(err, "Error marshalling data for Kafka")
	}

	//nolint:exhaustruct
	err = or.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(orderNum),
		Value: jsonData,
	})
	if err != nil {
		or.log.Error().Err(err).Msg("Error writing to kafka")

		return errors.Wrap(err, "Error writing to kafka")
	}

	or.log.Info().Interface("order", data).Msg("NewOrder sent to Kafka")

	return nil
}

func (or *Orders) ProcessOrders(ctx context.Context) {
	or.log.Info().Msg("Processing orders from Kafka")

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping message processing...")

			return
		default:
			msg, err := or.reader.FetchMessage(ctx)
			if err != nil {
				or.log.Error().Err(err).Msg("error reading message from kafka")

				continue
			}

			//nolint:exhaustruct
			data := KafkaOrder{}
			err = json.Unmarshal(msg.Value, &data)
			if err != nil {
				or.log.Error().Err(err).Msg("error unmarshalling order message from Kafka")

				return
			}

			or.log.Info().Interface("order", data).Msg("Processing data")

			operation := func() error {
				return or.checkOrderStatus(ctx, data)
			}

			err = utils.RetryOperation(ctx, operation)
			if err != nil {
				or.log.Error().Err(err).Str("orderId", data.OrderNum).Msg("error processing order")

				or.log.Info().Interface("order", data).Msg("Order can't be processed in Kafka, skipping")
				_ = or.reader.CommitMessages(ctx, msg)

				continue
			}

			_ = or.reader.CommitMessages(ctx, msg)
			or.log.Info().Interface("order", data).Msg("Order successfully processed data in Kafka")
		}
	}
}

// checkOrderStatus обновляет статус заказа, поддерживает ретраи через Kafka.
func (or *Orders) checkOrderStatus(ctx context.Context, message KafkaOrder) error {
	or.log.Info().Interface("OrderNum", message).Msg("Retrieving Order ID")

	result, err := or.sender.SendPostRequest(ctx, message.OrderNum)
	if err != nil {
		return err
	}

	err = or.storage.UpdateOrder(ctx, result, message.UserID)
	if err != nil {
		return errors.Wrap(err, "error updating order")
	}

	return nil
}
