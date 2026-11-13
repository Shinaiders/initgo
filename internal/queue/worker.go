package queue

import (
	"context"
	"encoding/json"
	"log"

	"goscrapper/internal/model"
)

type JobRunner interface {
	RunJob(ctx context.Context, jobID uint, req *model.ScrapeRequest) (*model.ScrapeJob, error)
}

type ScrapeMessagePayload struct {
	JobID   uint                `json:"job_id"`
	Request model.ScrapeRequest `json:"request"`
}

// StartWorker inicia o consumidor do RabbitMQ em background
func StartWorker(ctx context.Context, client *RabbitMQClient, runner JobRunner) {
	if client == nil {
		return
	}

	msgs, err := client.Consume()
	if err != nil {
		log.Printf("[RABBITMQ WORKER ERROR] Falha ao iniciar consumo da fila: %v", err)
		return
	}

	log.Println("[RABBITMQ WORKER] Worker iniciado e aguardando tarefas na fila...")

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("[RABBITMQ WORKER] Encerrando worker...")
				return
			case d, ok := <-msgs:
				if !ok {
					log.Println("[RABBITMQ WORKER] Canal de mensagens fechado.")
					return
				}

				var payload ScrapeMessagePayload
				if err := json.Unmarshal(d.Body, &payload); err != nil {
					log.Printf("[RABBITMQ WORKER] Erro ao deserializar mensagem: %v", err)
					_ = d.Nack(false, false) // Rejeita e descarta mensagem corrompida
					continue
				}

				log.Printf("[RABBITMQ WORKER] Processando Job ID %d para URL: %s", payload.JobID, payload.Request.URL)

				_, runErr := runner.RunJob(context.Background(), payload.JobID, &payload.Request)
				if runErr != nil {
					log.Printf("[RABBITMQ WORKER] Falha ao processar Job ID %d: %v", payload.JobID, runErr)
				} else {
					log.Printf("[RABBITMQ WORKER] Job ID %d processado com sucesso.", payload.JobID)
				}

				_ = d.Ack(false)
			}
		}
	}()
}
