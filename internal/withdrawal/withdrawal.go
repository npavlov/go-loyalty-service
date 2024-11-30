package withdrawal

import (
	"context"
	"encoding/json"
	"log"

	"github.com/npavlov/go-loyalty-service/internal/storage"
	"github.com/npavlov/go-loyalty-service/internal/utils"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
)

type KafkaWithdrawal struct {
	OrderNum string  `json:"orderNum"`
	UserId   string  `json:"UserId"`
	Sum      float64 `json:"sum"`
}

type Withdrawal struct {
	log     *zerolog.Logger
	writer  *kafka.Writer
	reader  *kafka.Reader
	storage *storage.DBStorage
}

func NewWithdrawal(writer *kafka.Writer, reader *kafka.Reader, storage *storage.DBStorage, log *zerolog.Logger) *Withdrawal {
	return &Withdrawal{
		log:     log,
		writer:  writer,
		reader:  reader,
		storage: storage,
	}
}

func (wd *Withdrawal) AddWithdrawal(ctx context.Context, orderNum string, userId string, sum float64) error {
	data := KafkaWithdrawal{
		OrderNum: orderNum,
		UserId:   userId,
		Sum:      sum,
	}

	jsonData, err := json.Marshal(data)

	if err != nil {
		wd.log.Error().Err(err).Msg("Error marshalling data for Kafka")

		return err
	}

	err = wd.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(orderNum),
		Value: jsonData,
	})
	if err != nil {
		return err
	}

	wd.log.Info().Interface("withdrawal", data).Msg("Withdrawal sent to Kafka")
	return nil
}

func (wd *Withdrawal) ProcessWithdrawal(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping message processing...")
			return
		default:
			msg, err := wd.reader.FetchMessage(ctx)
			if err != nil {
				wd.log.Error().Err(err).Msg("error reading message")

				continue
			}

			data := KafkaWithdrawal{}
			err = json.Unmarshal(msg.Value, &data)
			if err != nil {
				wd.log.Error().Err(err).Msg("error unmarshalling withdrawal message from Kafka")

				return
			}

			wd.log.Info().Interface("order", data).Msg("Processing data")

			operation := func() error {
				withdrawal, err := wd.storage.GetWithdrawal(ctx, data.OrderNum)
				if err != nil {
					return err
				}

				if withdrawal != nil {
					wd.log.Info().Interface("withdrawal", withdrawal).Msg("Withdrawal already exist, skipping")

					return nil
				}

				return wd.storage.MakeWithdrawn(ctx, data.UserId, data.OrderNum, data.Sum)
			}

			err = utils.RetryOperation(ctx, operation)
			if err != nil {
				wd.log.Error().Err(err).Str("orderId", data.OrderNum).Msg("error processing withdrawal")

				continue
			}

			_ = wd.reader.CommitMessages(ctx, msg)
			wd.log.Info().Interface("order", data).Msg("Withdrawal successfully processed data in Kafka")
		}
	}
}
