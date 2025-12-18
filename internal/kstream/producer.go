package kstream

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/segmentio/kafka-go"

	"gcr-backend/internal/model"
)

// kafkaWriter constructs a Kafka producer using segmentio/kafka-go library.
// kafka.Writer provides async message publishing with automatic batching and retries.
func kafkaWriter(topic string) *kafka.Writer {
	broker := getenv("KAFKA_BROKER", "kafka:9092")
	return &kafka.Writer{
		Addr:         kafka.TCP(broker),           // segmentio/kafka-go: TCP address for Kafka broker
		Topic:        topic,                        // Target Kafka topic name
		Balancer:     &kafka.LeastBytes{},         // segmentio/kafka-go: Partition selection strategy
		RequiredAcks: kafka.RequireOne,            // segmentio/kafka-go: Wait for leader ack only
		Async:        true,                        // segmentio/kafka-go: Non-blocking writes
		BatchBytes:   104857600,                   // segmentio/kafka-go: Max batch size (100MB) to handle large messages
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// PublishOnSearchIngest sends the raw on_search envelope to the ingest topic so
// downstream consumers (SchemaGate, Curated Writer) can process it.
func PublishOnSearchIngest(ctx context.Context, env *model.OnSearchEnvelope) error {
	w := kafkaWriter("catalog.ingest")
	defer w.Close()

	data, err := json.Marshal(env)
	if err != nil {
		return err
	}

	// segmentio/kafka-go: kafka.Message struct for publishing to Kafka topic.
	// Key is used for partitioning (same key → same partition for ordering).
	msg := kafka.Message{
		Key:   []byte(env.Context.TransactionID),
		Value: data,
		Time:  time.Now(),
	}
	// segmentio/kafka-go: WriteMessages publishes message to Kafka broker asynchronously.
	return w.WriteMessages(ctx, msg)
}

// PublishSearchRequest persists /search calls on a Kafka topic.
func PublishSearchRequest(ctx context.Context, payload any) error {
	w := kafkaWriter("catalog.search.requests")
	defer w.Close()

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Value: data,
		Time:  time.Now(),
	}
	return w.WriteMessages(ctx, msg)
}


