package queue

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"

	"github.com/npavlov/go-loyalty-service/internal/config"
)

type Queue struct {
	cfg *config.Config
	log *zerolog.Logger
}

func NewQueue(cfg *config.Config, log *zerolog.Logger) *Queue {
	return &Queue{
		cfg: cfg,
		log: log,
	}
}

func (cf *Queue) CreateKafkaWriter(topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(cf.cfg.Kafka),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},   // Load balances messages across partitions
		Async:        true,                  // Enable asynchronous writes
		BatchSize:    10,                    // Adjust batch size based on traffic
		BatchTimeout: 10 * time.Millisecond, // Max wait time for a batch
		Compression:  kafka.Snappy,
	}
}

func (cf *Queue) CreateKafkaReader(topic, groupID string) *kafka.Reader {
	brokers := []string{cf.cfg.Kafka}

	err := cf.checkKafkaAlive(cf.cfg.Kafka)
	if err != nil {
		cf.log.Error().Err(err).Msg("Kafka is dead")
	}

	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e3,
		MaxWait:  1 * time.Second,
	})
}

func (cf *Queue) CreateGroup(topic string) (*kafka.Writer, *kafka.Reader, func()) {
	orderWriter := cf.CreateKafkaWriter(topic)

	// Kafka Consumers
	orderReader := cf.CreateKafkaReader(topic, topic+"-group")

	closeFunc := func() {
		_ = orderWriter.Close()
		_ = orderReader.Close()
	}

	return orderWriter, orderReader, closeFunc
}

func (cf *Queue) checkKafkaAlive(broker string) error {
	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		return fmt.Errorf("unable to connect to Kafka broker: %w", err)
	}
	defer func(conn *kafka.Conn) {
		_ = conn.Close()
	}(conn)

	// Attempt to retrieve broker metadata to confirm the connection is live
	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("unable to retrieve controller info: %w", err)
	}
	cf.log.Info().Interface("controller", controller).Msg("Kafka connected to controller")

	return nil
}
