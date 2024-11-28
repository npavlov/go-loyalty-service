package orders

import (
	"context"
	"encoding/json"
	"log"

	"github.com/npavlov/go-loyalty-service/internal/config"
	"github.com/npavlov/go-loyalty-service/internal/storage"
	"github.com/npavlov/go-loyalty-service/internal/utils"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
)

type KafkaOrder struct {
	OrderNum string `json:"orderNum"`
	UserId   string `json:"UserId"`
}

type Orders struct {
	log     *zerolog.Logger
	writer  *kafka.Writer
	reader  *kafka.Reader
	storage *storage.DBStorage
	sender  *Sender
}

func NewOrders(writer *kafka.Writer, reader *kafka.Reader, log *zerolog.Logger) *Orders {
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

func (or *Orders) AddOrder(ctx context.Context, orderNum string, userId string) error {
	data := KafkaOrder{
		OrderNum: orderNum,
		UserId:   userId,
	}

	jsonData, err := json.Marshal(data)

	if err != nil {
		or.log.Error().Err(err).Msg("Error marshalling data for Kafka")

		return err
	}

	err = or.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(orderNum),
		Value: jsonData,
	})
	if err != nil {
		return err
	}

	or.log.Info().Msg("OrderId sent to Kafka")
	return nil
}

func (or *Orders) ProcessOrders(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping message processing...")
			return
		default:
			msg, err := or.reader.FetchMessage(ctx)
			if err != nil {
				or.log.Error().Err(err).Msg("error reading message")

				continue
			}

			data := KafkaOrder{}
			err = json.Unmarshal(msg.Value, &data)
			if err != nil {
				or.log.Error().Err(err).Msg("error unmarshalling message from Kafka")

				return
			}

			or.log.Info().Interface("order", data).Msg("Processing data")

			operation := func() error {
				return or.checkOrderStatus(ctx, data)
			}

			err = utils.RetryOperation(ctx, operation)
			if err != nil {
				or.log.Error().Err(err).Msg("error processing orders")
				_ = or.AddOrder(ctx, data.OrderNum, data.UserId)

				return
			}

			//_ = or.reader.CommitMessages(ctx, msg)
			or.log.Info().Interface("order", data).Msg("Successfully processed data in Kafka")
		}
	}
}

// checkOrderStatus обновляет статус заказа, поддерживает ретраи через Kafka.
func (or *Orders) checkOrderStatus(ctx context.Context, message KafkaOrder) error {
	or.log.Info().Interface("OrderNum", message).Msg("Retrieving Order Id")

	result, err := or.sender.SendPostRequest(ctx, message.OrderNum)

	if err != nil {
		return err
	}

	err = or.storage.UpdateOrder(ctx, result, message.UserId)

	if err != nil {
		return err
	}

	return nil
}
