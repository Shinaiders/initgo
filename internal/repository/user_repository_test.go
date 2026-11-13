package repository_test

import (
	"testing"

	"goscrapper/internal/model"
	"goscrapper/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// Cria banco SQLite em memória para testes rápidos
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("falha ao conectar no banco de teste em memória: %v", err)
	}

	// Executa AutoMigrate da tabela usuarios
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("falha ao rodar AutoMigrate no banco de teste: %v", err)
	}

	return db
}

func TestUserRepository_CRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewUserRepository(db)

	// 1. Teste de Criação e Hash de Senha Automático
	plainPassword := "senha123456"
	user := &model.User{
		Nome:  "Lucas Santana",
		Email: "lucas@teste.com",
		Senha: plainPassword,
		CPF:   "123.456.789-00",
	}

	err := repo.Create(user)
	if err != nil {
		t.Fatalf("erro ao criar usuário: %v", err)
	}

	if user.ID == 0 {
		t.Errorf("esperava que o ID fosse preenchido pelo GORM")
	}

	// A senha no banco NÃO pode ser o texto puro
	if user.Senha == plainPassword {
		t.Errorf("a senha deveria ter sido criptografada pelo BeforeSave Hook")
	}

	// Validação de senha com o método auxiliar
	if !user.CheckPassword(plainPassword) {
		t.Errorf("CheckPassword falhou para a senha correta")
	}
	if user.CheckPassword("senha_errada") {
		t.Errorf("CheckPassword deveria ter retornado false para senha incorreta")
	}

	// 2. Teste de Busca por Email
	foundByEmail, err := repo.GetByEmail("lucas@teste.com")
	if err != nil {
		t.Fatalf("erro ao buscar usuário por email: %v", err)
	}
	if foundByEmail.Nome != "Lucas Santana" {
		t.Errorf("esperava nome 'Lucas Santana', obteve '%s'", foundByEmail.Nome)
	}

	// 3. Teste de Busca por CPF
	foundByCPF, err := repo.GetByCPF("123.456.789-00")
	if err != nil {
		t.Fatalf("erro ao buscar usuário por CPF: %v", err)
	}
	if foundByCPF.ID != user.ID {
		t.Errorf("IDs não coincidem")
	}

	// 4. Teste de Unicidade de Email (uniqueIndex)
	duplicateUser := &model.User{
		Nome:  "Outro Lucas",
		Email: "lucas@teste.com", // Email duplicado
		Senha: "outrasenha123",
		CPF:   "999.888.777-66",
	}
	err = repo.Create(duplicateUser)
	if err == nil {
		t.Errorf("esperava erro de constraint UNIQUE para email duplicado, mas não ocorreu")
	}
}
