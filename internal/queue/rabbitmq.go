package queue

import (
	"context"
	"fmt"
	"log"
	"time"

	"goscrapper/internal/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQClient struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	queueName string
}

// InitRabbitMQ conecta ao RabbitMQ se ENABLE_RABBITMQ=true.
// Caso a variável esteja ativa e falhe, retorna erro obrigatório.
func InitRabbitMQ(cfg *config.Config) (*RabbitMQClient, error) {
	if !cfg.EnableRabbitMQ {
		log.Println("[RABBITMQ] Integração opcional com RabbitMQ está DESABILITADA (ENABLE_RABBITMQ=false)")
		return nil, nil
	}

	log.Printf("[RABBITMQ] Conectando ao RabbitMQ em %s (ENABLE_RABBITMQ=true)...", cfg.RabbitMQURL)
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar ao RabbitMQ (obrigatório com ENABLE_RABBITMQ=true): %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("falha ao abrir canal RabbitMQ: %w", err)
	}

	// Declara a fila durável
	q, err := ch.QueueDeclare(
		cfg.RabbitMQQueue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("falha ao declarar fila RabbitMQ (%s): %w", cfg.RabbitMQQueue, err)
	}

	log.Printf("[RABBITMQ] Conectado com sucesso. Fila pronta: %s\n", q.Name)

	return &RabbitMQClient{
		conn:      conn,
		channel:   ch,
		queueName: q.Name,
	}, nil
}

// Publish envia mensagem para a fila padrão
func (r *RabbitMQClient) Publish(ctx context.Context, body []byte) error {
	if r == nil || r.channel == nil {
		return nil
	}

	return r.channel.PublishWithContext(
		ctx,
		"",          // exchange default
		r.queueName, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
}

// Consume retorna o canal de consumo de mensagens
func (r *RabbitMQClient) Consume() (<-chan amqp.Delivery, error) {
	if r == nil || r.channel == nil {
		return nil, fmt.Errorf("cliente RabbitMQ não está inicializado")
	}

	return r.channel.Consume(
		r.queueName,
		"",    // consumer tag
		false, // auto-ack (false para ack manual seguro)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
}

// Close encerra canal e conexão
func (r *RabbitMQClient) Close() {
	if r != nil {
		if r.channel != nil {
			_ = r.channel.Close()
		}
		if r.conn != nil {
			_ = r.conn.Close()
		}
		log.Println("[RABBITMQ] Conexão com RabbitMQ encerrada.")
	}
}
