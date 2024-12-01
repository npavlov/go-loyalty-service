package queue

import (
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/npavlov/go-loyalty-service/internal/config"
)

type Queue struct {
	cfg *config.Config
}

func NewQueue(cfg *config.Config) *Queue {
	return &Queue{
		cfg: cfg,
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

	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e1, // 10KB
		MaxBytes: 10e3, // 10KB
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
