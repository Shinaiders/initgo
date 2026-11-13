# Tutorial Completo: Publicando e Consumindo Mensagens no RabbitMQ com Go 🐇

O **RabbitMQ** é um dos corretores de mensagens (_message brokers_) mais populares e confiáveis do mundo. Ele permite que você execute tarefas pesadas ou demoradas de forma **assíncrona e desacoplada**: a API recebe a requisição, coloca uma mensagem na fila e responde instantaneamente ao usuário, enquanto workers em background processam a tarefa no seu próprio ritmo.

Este guia ensina como conectar, declarar filas duráveis, publicar dados em JSON e consumir mensagens com controle de confirmação (_Ack/Nack_) e controle de vazão (_QoS_).

---

## 🧠 Conceitos Essenciais do RabbitMQ

Antes do código, entenda estes 5 conceitos fundamentais:

1. **Producer (Produtor):** Aplicação que envia mensagens para o RabbitMQ (ex: sua API HTTP).
2. **Queue (Fila):** O buffer que armazena as mensagens na memória ou disco até serem processadas.
3. **Consumer (Consumidor / Worker):** Aplicação que escuta a fila e processa as mensagens.
4. **Connection vs. Channel:**
   - **`Connection`:** Uma conexão TCP real entre o Go e o RabbitMQ. É pesada e deve ser aberta **uma única vez** no boot.
   - **`Channel`:** Uma sessão virtual leve dentro da conexão TCP. Quase todas as operações (declarar fila, publicar, consumir) acontecem dentro de canais.
5. **Acknowledgment (`Ack`):** O consumidor avisa o RabbitMQ: _"Recebi e processei essa mensagem com sucesso, pode apagá-la da fila"_. Se o worker morrer antes de dar o `Ack`, a mensagem volta para a fila automaticamente.

---

## 📦 Biblioteca Oficial em Go

A biblioteca oficial recomendada pela equipe do RabbitMQ é a [`github.com/rabbitmq/amqp091-go`](https://github.com/rabbitmq/amqp091-go):

```bash
go get github.com/rabbitmq/amqp091-go
```

---

## 🪜 Passo 1: Conexão e Declaração de Fila

Para publicar ou consumir, primeiro nos conectamos ao broker e declaramos uma fila. A declaração de fila é **idempotente** (se a fila já existir, nada acontece; se não existir, ela é criada).

```go
package main

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	url := "amqp://guest:guest@localhost:5672/"

	// 1. Abre a Conexão TCP
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatalf("Falha ao conectar no RabbitMQ: %v", err)
	}
	defer conn.Close()

	// 2. Abre um Canal dentro da Conexão
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Falha ao abrir canal: %v", err)
	}
	defer ch.Close()

	// 3. Declara a Fila
	q, err := ch.QueueDeclare(
		"minha_fila", // Nome da fila
		true,         // Durable: a fila sobrevive a reinicializações do RabbitMQ
		false,        // Delete when unused: deletar quando não houver consumidores?
		false,        // Exclusive: exclusiva apenas para esta conexão?
		false,        // No-wait: não esperar confirmação do broker?
		nil,          // Arguments adicionais
	)
	if err != nil {
		log.Fatalf("Falha ao declarar fila: %v", err)
	}

	fmt.Printf("Fila pronta para uso: %s\n", q.Name)
}
```

---

## 🪜 Passo 2: Como Publicar na Fila (Producer)

Ao publicar, serializamos nossos dados em **JSON** e enviamos com `DeliveryMode: Persistent` para que a mensagem seja gravada em disco caso o servidor do RabbitMQ reinicie.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Estrutura da mensagem que queremos trafegar
type PedidoScraping struct {
	ID        uint      `json:"id"`
	URL       string    `json:"url"`
	Timestamp time.Time `json:"timestamp"`
}

func publicarMensagem(ch *amqp.Channel, filaNome string, pedido PedidoScraping) error {
	// 1. Serializa a struct para bytes JSON
	body, err := json.Marshal(pedido)
	if err != nil {
		return fmt.Errorf("erro ao converter para JSON: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 2. Publica no canal
	err = ch.PublishWithContext(
		ctx,
		"",       // Exchange padrão (Default Direct Exchange)
		filaNome, // Routing Key: no exchange padrão, é o nome exato da fila
		false,    // Mandatory
		false,    // Immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent, // Mensagem persistida em disco
			ContentType:  "application/json",
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("erro ao publicar: %w", err)
	}

	log.Printf("[PRODUCER] Mensagem enviada para a fila: %+v\n", pedido)
	return nil
}
```

---

## 🪜 Passo 3: Como Consumir da Fila (Consumer / Worker)

No consumidor, você **NUNCA** deve usar `autoAck: true` em sistemas profissionais. Usamos `autoAck: false` para que o seu código confirme manualmente com `d.Ack(false)` apenas **depois** que o trabalho foi concluído com sucesso.

Também definimos o **`QoS` (Quality of Service)** para evitar que um único worker puxe todas as mensagens da fila de uma vez:

```go
package main

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type PedidoScraping struct {
	ID  uint   `json:"id"`
	URL string `json:"url"`
}

func iniciarConsumidor(ch *amqp.Channel, filaNome string) {
	// 1. QoS (Fair Dispatch): entrega no máximo 1 mensagem por vez para este worker
	// Ele só receberá a próxima mensagem quando der Ack na atual!
	err := ch.Qos(
		1,     // Prefetch count: 1 mensagem por vez
		0,     // Prefetch size
		false, // Global
	)
	if err != nil {
		log.Fatalf("Falha ao configurar QoS: %v", err)
	}

	// 2. Registra o consumidor na fila
	msgs, err := ch.Consume(
		filaNome,
		"",    // Consumer tag (vazio = gerado automaticamente)
		false, // Auto-Ack: FALSE (obrigatório para segurança!)
		false, // Exclusive
		false, // No-local
		false, // No-wait
		nil,   // Args
	)
	if err != nil {
		log.Fatalf("Falha ao registrar consumidor: %v", err)
	}

	log.Println("[CONSUMER] Worker conectado. Aguardando mensagens... Para sair aperte CTRL+C")

	// 3. Loop infinito lendo mensagens do canal Go (<-chan amqp.Delivery)
	for d := range msgs {
		log.Printf("[CONSUMER] Mensagem recebida! Tamanho: %d bytes", len(d.Body))

		var pedido PedidoScraping
		if err := json.Unmarshal(d.Body, &pedido); err != nil {
			log.Printf("[ERRO] JSON corrompido: %v", err)
			// d.Nack(multiple=false, requeue=false) -> Descarta a mensagem inválida
			_ = d.Nack(false, false)
			continue
		}

		// --- Processamento Real da Tarefa ---
		log.Printf("[PROCESSANDO] Raspando a URL: %s (ID: %d)...", pedido.URL, pedido.ID)

		// Simulação de tarefa demorada:
		sucesso := processarTarefa(pedido)

		if sucesso {
			// d.Ack(multiple=false) -> Confirma sucesso ao RabbitMQ
			log.Printf("[SUCESSO] Tarefa %d concluída!", pedido.ID)
			_ = d.Ack(false)
		} else {
			// d.Nack(multiple=false, requeue=true) -> Devolve para a fila para tentar novamente
			log.Printf("[FALHA] Erro ao processar. Devolvendo para a fila...")
			_ = d.Nack(false, true)
		}
	}
}

func processarTarefa(p PedidoScraping) bool {
	// Aqui você roda o Playwright, envia e-mail, etc.
	return true
}
```

---

## 🚦 A Trindade de Confirmação: `Ack`, `Nack` e `Reject`

| Método                 | Para que serve                                                    | O que acontece com a mensagem?                              |
| :--------------------- | :---------------------------------------------------------------- | :---------------------------------------------------------- |
| `d.Ack(false)`         | Sucesso absoluto.                                                 | É removida definitivamente da fila.                         |
| `d.Nack(false, true)`  | Erro temporário (ex: timeout de rede ou banco fora).              | É devolvida à fila (`requeue: true`) para ser reprocessada. |
| `d.Nack(false, false)` | Erro fatal irrecuperável (ex: JSON corrompido ou ID inexistente). | É descartada da fila (ou vai para uma _Dead Letter Queue_). |

---

## 🏛️ Como o RabbitMQ está Integrado no Nosso Projeto

Neste projeto de automação, o RabbitMQ já está 100% plugado e pronto para uso:

1. **Cliente e Canais ([internal/queue/rabbitmq.go](file:///Users/lucassantana/Projetos/goscrapper/internal/queue/rabbitmq.go))**:
   - Cria a conexão e o canal se `ENABLE_RABBITMQ=true`.
   - Método `Publish(ctx, jsonBytes)` para enfileirar mensagens persistentes.
2. **Worker Concorrente ([internal/queue/worker.go](file:///Users/lucassantana/Projetos/goscrapper/internal/queue/worker.go))**:
   - Roda em uma goroutine no boot da aplicação consumindo tarefas assíncronas de scraping.
3. **Disparo no Serviço ([internal/service/scrape_service.go](file:///Users/lucassantana/Projetos/goscrapper/internal/service/scrape_service.go))**:
   - Quando você dispara um scraping com `"async": true`, o serviço publica a tarefa no RabbitMQ e devolve o status `PENDING` para o cliente em milissegundos.

---

## 🧪 Como Testar no Seu Computador Agora

### 1. Suba o RabbitMQ com Docker:

```bash
make deps-up
```

### 2. Acesse o Painel de Controle Web:

Abra no navegador: **[http://localhost:15672](http://localhost:15672)**

- **Usuário:** `guest`
- **Senha:** `guest`

Na aba **Queues**, você verá a fila `scrape_tasks` e poderá acompanhar em tempo real os gráficos de mensagens entrando, sendo processadas pelos workers e saindo com Ack!
