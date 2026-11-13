package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User representa a entidade da tabela 'usuarios' no banco de dados
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Nome      string         `gorm:"type:varchar(150);not null" json:"nome"`
	Email     string         `gorm:"type:varchar(150);uniqueIndex;not null" json:"email"`
	Senha     string         `gorm:"type:varchar(255);not null" json:"-"` // Oculta a senha em respostas JSON
	CPF       string         `gorm:"type:varchar(14);uniqueIndex;not null" json:"cpf"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName define o nome explícito da tabela no banco
func (User) TableName() string {
	return "usuarios"
}

// BeforeSave é um Hook do GORM chamado antes de criar ou atualizar um usuário.
// Garante que a senha seja salva sempre criptografada com Bcrypt.
func (u *User) BeforeSave(tx *gorm.DB) error {
	// Se a senha já for um hash bcrypt (tamanho padrão de 60 chars), não gera novamente
	if len(u.Senha) == 60 {
		return nil
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(u.Senha), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.Senha = string(hashed)
	return nil
}

// CheckPassword valida se uma senha informada em texto puro confere com o hash salvo
func (u *User) CheckPassword(plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Senha), []byte(plainPassword))
	return err == nil
}

// CreateUserRequest DTO para entrada de cadastro de usuário
type CreateUserRequest struct {
	Nome  string `json:"nome" binding:"required,min=2,max=150"`
	Email string `json:"email" binding:"required,email"`
	Senha string `json:"senha" binding:"required,min=6,max=50"`
	CPF   string `json:"cpf" binding:"required,min=11,max=14"`
}

// UserResponse DTO seguro de saída (nunca expõe a senha)
type UserResponse struct {
	ID        uint      `json:"id"`
	Nome      string    `json:"nome"`
	Email     string    `json:"email"`
	CPF       string    `json:"cpf"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converte a entidade User em UserResponse
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Nome:      u.Nome,
		Email:     u.Email,
		CPF:       u.CPF,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
