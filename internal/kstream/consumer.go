package kstream

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"

	"gcr-backend/internal/curated"
	"gcr-backend/internal/model"
	"gcr-backend/internal/rejections"
	"gcr-backend/internal/schemagate"
)

// KafkaReader creates a Kafka consumer using segmentio/kafka-go library.
// kafka.Reader provides consumer group functionality with automatic offset management.
func KafkaReader(topic, groupID string) *kafka.Reader {
	broker := getenv("KAFKA_BROKER", "kafka:9092")
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{broker},          // segmentio/kafka-go: Kafka broker addresses
		Topic:          topic,                     // segmentio/kafka-go: Topic to consume from
		GroupID:        groupID,                   // segmentio/kafka-go: Consumer group ID (enables load balancing)
		MinBytes:       10e3,                      // segmentio/kafka-go: Min bytes to fetch per request (10KB)
		MaxBytes:       104857600,                 // segmentio/kafka-go: Max bytes per message (100MB) to handle large payloads
		CommitInterval: time.Second,               // segmentio/kafka-go: Auto-commit interval for offsets
	})
}

// kafkaReader is an alias for internal use.
func kafkaReader(topic, groupID string) *kafka.Reader {
	return KafkaReader(topic, groupID)
}

// ConsumeIngestTopic runs SchemaGate consumer that reads from catalog.ingest,
// validates providers, writes rejects, and forwards valid rows to Curated Writer.
func ConsumeIngestTopic(ctx context.Context) error {
	reader := kafkaReader("catalog.ingest", "schemagate-group")
	defer reader.Close()

	log.Println("SchemaGate: consuming from catalog.ingest")

	for {
		// segmentio/kafka-go: ReadMessage blocks until a message is available from Kafka topic.
		// Automatically handles consumer group coordination and offset commits.
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			return err
		}

		var env model.OnSearchEnvelope
		if err := json.Unmarshal(msg.Value, &env); err != nil {
			log.Printf("SchemaGate: failed to unmarshal: %v", err)
			continue
		}

		// Step-2: Provider/Item validation with partial acceptance
		validProviders, rejectionsList := schemagate.ProcessCatalog(ctx, &env)

		// Write rejections to durable store
		envMeta := map[string]string{
			"transaction_id": env.Context.TransactionID,
			"message_id":     env.Context.MessageID,
		}
		for _, rej := range rejectionsList {
			_ = rejections.WriteRejection(ctx, envMeta, rej)
		}

		// Forward valid providers to Curated Writer
		if len(validProviders) > 0 {
			events, err := curated.WriteValidProviders(ctx, &env, validProviders)
			if err != nil {
				log.Printf("Curated Writer: error: %v", err)
				continue
			}

			// Publish CatalogAccepted events
			for _, evt := range events {
				if err := PublishCatalogAccepted(ctx, evt); err != nil {
					log.Printf("Failed to publish CatalogAccepted: %v", err)
				}
			}
		}
	}
}

// PublishCatalogAccepted publishes a CatalogAccepted event to topic.catalog.accepted.
func PublishCatalogAccepted(ctx context.Context, evt model.CatalogAccepted) error {
	w := kafkaWriter("catalog.accepted")
	defer w.Close()

	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(evt.SellerID + ":" + evt.City + ":" + evt.Category),
		Value: data,
		Time:  time.Now(),
	}
	return w.WriteMessages(ctx, msg)
}

