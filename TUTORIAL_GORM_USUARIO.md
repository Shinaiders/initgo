# Tutorial: Criando a Tabela de Usuários com GORM em Go 📚

Este guia ensina passo a passo como declarar, migrar, manipular e proteger uma tabela de **Usuários** (`usuarios`) usando o **GORM** no ecossistema Go.

---

## 🎯 O que vamos construir?

Uma tabela com as seguintes colunas e regras de negócio:
- **`id`**: Chave primária auto-incrementada (`PRIMARY KEY`).
- **`nome`**: Nome do usuário (`VARCHAR(150)`, não nulo).
- **`email`**: E-mail único (`VARCHAR(150)`, não nulo, índice único).
- **`senha`**: Hash criptografado da senha (`VARCHAR(255)`, nunca retornado em JSON).
- **`cpf`**: CPF único (`VARCHAR(14)`, não nulo, índice único).
- **`created_at`** e **`updated_at`**: Timestamps automáticos gerenciados pelo GORM.
- **`deleted_at`**: Soft delete (exclusão lógica sem perda de dados).

---

## 🪜 Passo 1: Definir o Modelo (`Struct`)

No GORM, você não escreve `CREATE TABLE` manualmente. Você cria uma `struct` em Go decorada com tags de metadados:

Arquivo: [`internal/model/user.go`](file:///Users/lucassantana/Projetos/goscrapper/internal/model/user.go)

```go
package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Nome      string         `gorm:"type:varchar(150);not null" json:"nome"`
	Email     string         `gorm:"type:varchar(150);uniqueIndex;not null" json:"email"`
	Senha     string         `gorm:"type:varchar(255);not null" json:"-"`
	CPF       string         `gorm:"type:varchar(14);uniqueIndex;not null" json:"cpf"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName força o nome da tabela no banco a ser "usuarios" (em vez do plural padrão em inglês "users")
func (User) TableName() string {
	return "usuarios"
}
```

### 🔍 Anatomia das Tags:
| Tag | Propósito |
| :--- | :--- |
| `gorm:"primaryKey"` | Define o campo como identificador único auto-incrementado. |
| `gorm:"type:varchar(150)"` | Especifica o tipo exato no banco de dados. |
| `gorm:"not null"` | Impede que a coluna aceite valores nulos (`NOT NULL`). |
| `gorm:"uniqueIndex"` | Cria um índice exclusivo no banco. Impede cadastros duplicados de e-mail ou CPF. |
| `json:"-"` | **Segurança essencial:** impede que a senha seja exposta quando o objeto for convertido em JSON na API. |
| `gorm.DeletedAt` | Ativa o recurso de **Soft Delete**: ao deletar, o GORM apenas preenche a data atual em `deleted_at`. Consultas normais (`Find`, `First`) ignoram registros "deletados" automaticamente. |

---

## 🔐 Passo 2: Criptografia Automática de Senha (Hooks do GORM)

Nunca salve senhas em texto puro no banco. O GORM possui **Hooks de Ciclo de Vida** que são executados antes ou depois de operações no banco.

Adicionamos o hook `BeforeSave`:

```go
// BeforeSave é executado automaticamente pelo GORM antes de qualquer INSERT ou UPDATE
func (u *User) BeforeSave(tx *gorm.DB) error {
	// Se a senha já tiver 60 caracteres, já é um hash Bcrypt válido
	if len(u.Senha) == 60 {
		return nil
	}

	// Gera o hash com custo padrão (10 rounds)
	hashed, err := bcrypt.GenerateFromPassword([]byte(u.Senha), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.Senha = string(hashed)
	return nil
}

// CheckPassword compara uma senha fornecida com o hash gravado
func (u *User) CheckPassword(plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Senha), []byte(plainPassword))
	return err == nil
}
```

---

## 🗄️ Passo 3: Executar a Migração Automática (`AutoMigrate`)

No arquivo de inicialização do banco ([`internal/database/database.go`](file:///Users/lucassantana/Projetos/goscrapper/internal/database/database.go)), registramos o modelo:

```go
func runMigrations(db *gorm.DB) error {
	log.Println("[DATABASE] Executando auto-migrations...")
	return db.AutoMigrate(
		&model.ScrapeJob{},
		&model.User{}, // <-- GORM inspeciona a struct e cria a tabela se ela não existir
	)
}
```

> 💡 **O que o GORM gera no banco (SQL equivalente):**
> ```sql
> CREATE TABLE IF NOT EXISTS usuarios (
>     id BIGSERIAL PRIMARY KEY,
>     nome VARCHAR(150) NOT NULL,
>     email VARCHAR(150) NOT NULL,
>     senha VARCHAR(255) NOT NULL,
>     cpf VARCHAR(14) NOT NULL,
>     created_at TIMESTAMP WITH TIME ZONE,
>     updated_at TIMESTAMP WITH TIME ZONE,
>     deleted_at TIMESTAMP WITH TIME ZONE
> );
> CREATE UNIQUE INDEX IF NOT EXISTS idx_usuarios_email ON usuarios(email);
> CREATE UNIQUE INDEX IF NOT EXISTS idx_usuarios_cpf ON usuarios(cpf);
> CREATE INDEX IF NOT EXISTS idx_usuarios_deleted_at ON usuarios(deleted_at);
> ```

---

## 🛠️ Passo 4: Operações no Banco com Repository (`CRUD`)

No padrão Clean Architecture, isolamos o acesso a dados no arquivo [`internal/repository/user_repository.go`](file:///Users/lucassantana/Projetos/goscrapper/internal/repository/user_repository.go):

### 1. Inserir Usuário (`Create`)
```go
func (r *userRepository) Create(user *model.User) error {
    return r.db.Create(user).Error
}
```
*O hook `BeforeSave` dispara automaticamente antes do INSERT.*

### 2. Buscar por ID (`First`)
```go
func (r *userRepository) GetByID(id uint) (*model.User, error) {
    var user model.User
    if err := r.db.First(&user, id).Error; err != nil {
        return nil, err // Retorna gorm.ErrRecordNotFound se não existir
    }
    return &user, nil
}
```

### 3. Buscar com Filtro (`Where`)
```go
func (r *userRepository) GetByEmail(email string) (*model.User, error) {
    var user model.User
    err := r.db.Where("email = ?", email).First(&user).Error
    return &user, err
}
```

### 4. Atualizar Registro (`Save`)
```go
func (r *userRepository) Update(user *model.User) error {
    return r.db.Save(user).Error
}
```

### 5. Excluir (`Delete` com Soft Delete)
```go
func (r *userRepository) Delete(id uint) error {
    return r.db.Delete(&model.User{}, id).Error
}
```

---

## 🧪 Passo 5: Como Testar com SQLite em Memória

Você pode rodar os testes unitários criados no projeto para ver a tabela sendo criada e testada:

```bash
go test -v ./internal/repository/...
```

O teste valida automaticamente:
1. A criação da tabela e preenchimento do `ID`.
2. O hash da senha via `bcrypt`.
3. A busca por e-mail e CPF.
4. A rejeição de e-mails duplicados com erro `UNIQUE constraint failed`.
