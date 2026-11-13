# Tutorial Definitivo: Como Fazer "Fetch" (Requisições HTTP) em Go 🌐

Se você vem do ecossistema JavaScript/TypeScript, está acostumado com a função global `fetch()`. Em **Go**, não precisamos de nenhuma biblioteca externa como Axios ou Node-Fetch: a biblioteca padrão [`net/http`](https://pkg.go.dev/net/http) é uma das mais poderosas, performáticas e completas do mercado.

Este guia ensina desde o básico até as práticas essenciais para código de nível sênior em produção.

---

## 🧭 Resumo Rápido: Comparação JS vs. Go

| Operação          | JavaScript / TypeScript                      | Go (`net/http`)                                      |
| :---------------- | :------------------------------------------- | :--------------------------------------------------- |
| **GET Simples**   | `const res = await fetch(url)`               | `res, err := http.Get(url)`                          |
| **Ler JSON**      | `const data = await res.json()`              | `json.NewDecoder(res.Body).Decode(&data)`            |
| **POST com Body** | `fetch(url, { method: 'POST', body })`       | `http.NewRequestWithContext(ctx, "POST", url, body)` |
| **Headers**       | `headers: { 'Authorization': 'Bearer ...' }` | `req.Header.Set("Authorization", "Bearer ...")`      |
| **Timeout**       | `AbortController`                            | `context.WithTimeout(ctx, 10*time.Second)`           |

---

## ⚠️ A Regra de Ouro do Go: `defer res.Body.Close()`

> [!CAUTION]
> Em Go, se você não fechar o `res.Body` após uma requisição, a conexão TCP subjacente não será reutilizada e sua aplicação sofrerá com **vazamento de memória e exaustão de conexões (File Descriptors)**.
> **Sempre** coloque `defer res.Body.Close()` logo após verificar que `err == nil`.

---

## 1. O "Fetch" Básico (GET Simples)

O exemplo mais direto para buscar texto, HTML ou dados brutos:

```go
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	url := "https://jsonplaceholder.typicode.com/posts/1"

	// 1. Executa a requisição GET
	res, err := http.Get(url)
	if err != nil {
		log.Fatalf("Erro ao fazer requisição: %v", err)
	}
	defer res.Body.Close() // <-- REGRA DE OURO

	// 2. Valida o código HTTP
	if res.StatusCode != http.StatusOK {
		log.Fatalf("Status inesperado: %d", res.StatusCode)
	}

	// 3. Lê o corpo da resposta como bytes
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("Erro ao ler body: %v", err)
	}

	fmt.Println("Resposta bruta:")
	fmt.Println(string(bodyBytes))
}
```

---

## 2. Consumindo APIs REST e Fazendo Parse de JSON

Em vez de converter todo o corpo em `string` e depois fazer `json.Unmarshal`, a forma idiomática e mais eficiente em Go é usar o **`json.NewDecoder`**, que faz o streaming do JSON diretamente do body:

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Estrutura mapeando a resposta da API do ViaCEP
type Endereco struct {
	CEP         string `json:"cep"`
	Logradouro  string `json:"logradouro"`
	Bairro      string `json:"bairro"`
	Localidade  string `json:"localidade"`
	UF          string `json:"uf"`
}

func main() {
	cep := "01001000"
	url := fmt.Sprintf("https://viacep.com.br/ws/%s/json/", cep)

	// Criamos um client com Timeout de 5 segundos
	client := &http.Client{Timeout: 5 * time.Second}

	res, err := client.Get(url)
	if err != nil {
		log.Fatalf("Falha na chamada: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Fatalf("Erro HTTP: status %d", res.StatusCode)
	}

	// Decodifica o JSON direto na struct
	var endereco Endereco
	if err := json.NewDecoder(res.Body).Decode(&endereco); err != nil {
		log.Fatalf("Erro ao decodificar JSON: %v", err)
	}

	fmt.Printf("Endereço encontrado: %s, %s - %s/%s\n",
		endereco.Logradouro, endereco.Bairro, endereco.Localidade, endereco.UF)
}
```

---

## 3. Enviando POST com Body JSON e Headers Customizados

Quando você precisa enviar dados para a API (`POST`, `PUT`, `PATCH`), enviar tokens de autenticação (`Authorization: Bearer ...`) ou definir `Content-Type`:

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type NovoPost struct {
	Titulo string `json:"title"`
	Corpo  string `json:"body"`
	UserID int    `json:"userId"`
}

type RespostaPost struct {
	ID     int    `json:"id"`
	Titulo string `json:"title"`
}

func main() {
	url := "https://jsonplaceholder.typicode.com/posts"

	// 1. Dados que serão enviados no corpo
	payload := NovoPost{
		Titulo: "Aprenda Go",
		Corpo:  "Guia definitivo de HTTP no Go",
		UserID: 1,
	}

	// 2. Converte struct para bytes JSON
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Erro ao serializar JSON: %v", err)
	}

	// 3. Cria a requisição HTTP com bytes.NewBuffer
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		log.Fatalf("Erro ao criar requisição: %v", err)
	}

	// 4. Adiciona os Headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer meu_token_secreto_123")
	req.Header.Set("User-Agent", "GoScrapper-Bot/1.0")

	// 5. Executa a requisição com o Client
	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("Falha ao enviar POST: %v", err)
	}
	defer res.Body.Close()

	// 6. Trata a resposta
	var resposta RespostaPost
	if err := json.NewDecoder(res.Body).Decode(&resposta); err != nil {
		log.Fatalf("Erro ao decodificar resposta: %v", err)
	}

	fmt.Printf("Post criado com sucesso! ID: %d - Título: %s\n", resposta.ID, resposta.Titulo)
}
```

---

## 4. O Padrão Sênior: Usando `context.Context` para Cancelamento e Timeout

No Go moderno, você sempre deve passar um `context.Context`. Isso permite que a requisição seja cancelada se o usuário fechar a aba, se o servidor receber um sinal de desligamento (_graceful shutdown_) ou se estourar o tempo limite:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

func buscarComTimeout(url string) error {
	// Timeout limite de 2 segundos para a operação inteira
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Cria requisição vinculada ao contexto
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		// Se o tempo estourar, o erro será context.DeadlineExceeded
		return fmt.Errorf("requisição cancelada ou expirada: %w", err)
	}
	defer res.Body.Close()

	fmt.Printf("Sucesso! Status: %d\n", res.StatusCode)
	return nil
}
```

---

## 5. Boas Práticas Essenciais de Produção

### 1. Nunca use `http.DefaultClient` sem Timeout em Produção

O cliente padrão do Go (`http.Get`) não tem timeout (`Timeout: 0`). Se o servidor de destino travar, sua goroutine ficará pendurada para sempre, derrubando sua aplicação por vazamento de memória.

```go
// ❌ Evite em produção:
http.Get("https://api-lenta.com")

// ✅ Correto:
client := &http.Client{
    Timeout: 10 * time.Second,
}
```

### 2. Reutilize o `*http.Client` (Connection Pooling)

O `http.Client` em Go gerencia um pool de conexões TCP (_Keep-Alive_) de forma concorrente e segura para goroutines (_thread-safe_). **Não crie um `http.Client` a cada função**. Crie um único cliente e passe-o por injeção de dependência:

```go
type MeuServico struct {
    httpClient *http.Client
}

func NovoServico() *MeuServico {
    return &MeuServico{
        httpClient: &http.Client{
            Timeout: 15 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 20,
                IdleConnTimeout:     90 * time.Second,
            },
        },
    }
}
```

---

## 6. O "Fetch" dos Sonhos em Go (Genérico e Reutilizável)

Abaixo está uma função utilitária com **Generics do Go (1.18+)** que você pode colocar em `pkg/fetcher/fetcher.go` e usar em todo o seu projeto exatamente como o `fetch()` do JavaScript:

```go
package fetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var defaultClient = &http.Client{
	Timeout: 15 * time.Second,
}

// Fetch envia uma requisição e deserializa a resposta JSON diretamente no tipo genérico T
func Fetch[T any](ctx context.Context, method, url string, body any) (*T, error) {
	var bodyReader io.Reader

	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("erro ao codificar body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("erro ao instanciar request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := defaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha de rede: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("status HTTP de erro: %d", res.StatusCode)
	}

	var resultado T
	if err := json.NewDecoder(res.Body).Decode(&resultado); err != nil {
		return nil, fmt.Errorf("falha ao decodificar JSON de retorno: %w", err)
	}

	return &resultado, nil
}
```

### Exemplo de uso dessa função:

```go
type Usuario struct {
    ID   int    `json:"id"`
    Nome string `json:"name"`
}

func main() {
    ctx := context.Background()
    usuario, err := fetcher.Fetch[Usuario](ctx, "GET", "https://api.com/users/1", nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(usuario.Nome)
}
```
