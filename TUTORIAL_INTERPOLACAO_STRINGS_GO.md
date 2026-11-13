# Tutorial: Interpolação e Formatação de Strings em Go 🧵

No JavaScript, você usa **Template Literals** (crases e `${variavel}`) para interpolar variáveis dentro de um texto:

```javascript
// JavaScript
const nome = "Lucas";
const idade = 25;
const saldo = 149.9;

const mensagem = `Olá ${nome}, você tem ${idade} anos e seu saldo é R$ ${saldo.toFixed(2)}.`;
```

Em **Go não existem Template Literals com crase desse mesmo jeito** (em Go, as crases criam _Raw String Literals_, úteis para textos multilinha ou regex, mas sem interpolação direta).

Para fazer isso em Go, utilizamos o pacote nativo **`fmt`** com a função **`fmt.Sprintf`**, que é a forma padrão e mais poderosa do ecossistema.

---

## 🧭 Tabela de Equivalência Rápida

| No JavaScript                          | No Go                                  |
| :------------------------------------- | :------------------------------------- |
| `` `Olá ${nome}` ``                    | `fmt.Sprintf("Olá %s", nome)`          |
| `` `Idade: ${idade}` ``                | `fmt.Sprintf("Idade: %d", idade)`      |
| `` `Preço: R$ ${preco.toFixed(2)}` ``  | `fmt.Sprintf("Preço: R$ %.2f", preco)` |
| `` `Ativo: ${ativo}` ``                | `fmt.Sprintf("Ativo: %t", ativo)`      |
| `` `Dados: ${JSON.stringify(user)}` `` | `fmt.Sprintf("Dados: %+v", user)`      |
| `"a" + "b"`                            | `"a" + "b"`                            |
| `[a, b].join(", ")`                    | `strings.Join([]string{a, b}, ", ")`   |

---

## 1. A Forma Padrão: `fmt.Sprintf`

A função `fmt.Sprintf` formata a string e **retorna o resultado como uma nova variável** (diferente de `fmt.Printf`, que apenas imprime na tela).

```go
package main

import "fmt"

func main() {
	nome := "Lucas"
	idade := 25
	saldo := 149.90

	// O equivalente exato a `Olá ${nome}, você tem ${idade} anos e saldo de R$ ${saldo}`
	mensagem := fmt.Sprintf("Olá %s, você tem %d anos e saldo de R$ %.2f.", nome, idade, saldo)

	fmt.Println(mensagem)
	// Saída: Olá Lucas, você tem 25 anos e saldo de R$ 149.90.
}
```

---

## 🔍 O "Alfabeto" dos Verbos de Formatação em Go

Os símbolos iniciados com `%` são chamados de **Format Verbs** (verbos de formatação):

|   Verbo    | Para que serve                                   | Exemplo                          | Saída                 |
| :--------: | :----------------------------------------------- | :------------------------------- | :-------------------- |
|  **`%s`**  | Strings de texto                                 | `fmt.Sprintf("Oi, %s", "Lucas")` | `"Oi, Lucas"`         |
|  **`%d`**  | Números Inteiros (decimais)                      | `fmt.Sprintf("Qtd: %d", 42)`     | `"Qtd: 42"`           |
|  **`%f`**  | Números decimais (Float)                         | `fmt.Sprintf("Valor: %f", 10.5)` | `"Valor: 10.500000"`  |
| **`%.2f`** | Float com **2 casas decimais**                   | `fmt.Sprintf("R$ %.2f", 10.5)`   | `"R$ 10.50"`          |
|  **`%t`**  | Booleanos (`true` ou `false`)                    | `fmt.Sprintf("Ativo: %t", true)` | `"Ativo: true"`       |
|  **`%v`**  | **Qualquer valor** (_Value_ genérico)            | `fmt.Sprintf("Algo: %v", 123)`   | `"Algo: 123"`         |
| **`%+v`**  | **Structs com nome dos campos** (Top para logs!) | `fmt.Sprintf("%+v", usuario)`    | `"{ID:1 Nome:Lucas}"` |
|  **`%T`**  | Mostra o **Tipo** da variável                    | `fmt.Sprintf("Tipo: %T", 42)`    | `"Tipo: int"`         |
|  **`%%`**  | O caractere `%` literal                          | `fmt.Sprintf("Taxa: %d%%", 15)`  | `"Taxa: 15%"`         |

---

## 2. O Verbo Mágico `%v` e `%+v` (Para Structs e Depuração)

Se você não quer ficar lembrando se a variável é `%s` ou `%d`, você pode usar `%v` (ele aceita qualquer tipo).

E para inspecionar objetos/structs (como o `console.log(objeto)` do JS), use **`%+v`**:

```go
package main

import "fmt"

type Usuario struct {
	Nome  string
	Email string
	Idade int
}

func main() {
	user := Usuario{
		Nome:  "Lucas Santana",
		Email: "lucas@exemplo.com",
		Idade: 25,
	}

	// %+v imprime o nome das chaves e os valores
	fmt.Printf("Dados do usuário: %+v\n", user)
	// Saída: Dados do usuário: {Nome:Lucas Santana Email:lucas@exemplo.com Idade:25}
}
```

---

## 3. Concatenação com o Operador `+`

Você pode concatenar strings usando o operador `+`, mas **apenas se todas as variáveis já forem strings**:

```go
primeiroNome := "Lucas"
sobrenome := "Santana"

nomeCompleto := primeiroNome + " " + sobrenome
```

### Se envolver números, use `strconv`:

No JavaScript você pode fazer `"Idade: " + 25` e ele converte sozinho.  
Em Go, como a linguagem é fortemente tipada, você precisa converter números antes:

```go
package main

import (
	"fmt"
	"strconv"
)

func main() {
	idade := 25

	// strconv.Itoa (Integer to ASCII)
	mensagem := "Idade: " + strconv.Itoa(idade)

	fmt.Println(mensagem) // "Idade: 25"
}
```

> 💡 **Dica:** Para misturar texto e números, `fmt.Sprintf("Idade: %d", idade)` é quase sempre mais limpo do que usar `strconv.Itoa`.

---

## 4. Unindo Arrays/Slices: `strings.Join`

No JavaScript você faz: `['São Paulo', 'Rio', 'Curitiba'].join(' -> ')`.  
No Go, você usa **`strings.Join`**:

```go
package main

import (
	"fmt"
	"strings"
)

func main() {
	cidades := []string{"São Paulo", "Rio de Janeiro", "Curitiba"}

	resultado := strings.Join(cidades, " -> ")
	fmt.Println(resultado)
	// Saída: São Paulo -> Rio de Janeiro -> Curitiba
}
```

---

## 5. Performance em Loops: `strings.Builder`

Se você precisa montar uma string gigante dentro de um loop (por exemplo, gerando um CSV ou relatório com milhares de linhas), **não use `+` nem `fmt.Sprintf`**, pois eles alocam memória a cada iteração.

Use o **`strings.Builder`**:

```go
package main

import (
	"fmt"
	"strings"
)

func main() {
	var builder strings.Builder

	for i := 1; i <= 5; i++ {
		// Escreve sem criar novas strings na memória
		builder.WriteString(fmt.Sprintf("Linha %d\n", i))
	}

	textoFinal := builder.String()
	fmt.Print(textoFinal)
}
```

---

## 🎯 Exemplos Práticos do Dia a Dia

### Montando URLs com parâmetros:

```go
baseURL := "https://api.github.com/users"
username := "lucassantana"
url := fmt.Sprintf("%s/%s/repos?page=%d&limit=%d", baseURL, username, 1, 10)
// "https://api.github.com/users/lucassantana/repos?page=1&limit=10"
```

### Formatando Moeda / Preços:

```go
preco := 1299.5
moeda := fmt.Sprintf("R$ %.2f", preco)
// "R$ 1299.50"
```

### Formatando Logs com Data e Hora:

```go
codigoErro := 404
rota := "/api/v1/scrapes"
logMsg := fmt.Sprintf("[%s] Erro %d ao acessar %s", "WARN", codigoErro, rota)
// "[WARN] Erro 404 ao acessar /api/v1/scrapes"
```
