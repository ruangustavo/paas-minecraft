package repository

import (
	"fmt"
	"paas-minecraft/internal/modules/server/model"
	"paas-minecraft/internal/shared/database"
)

type ServerRepository struct{}

func NewServerRepository() *ServerRepository {
	return &ServerRepository{}
}

func (sr *ServerRepository) Create(name string) (*model.Server, error) {
	server := &model.Server{
		Name: name,
	}

	db := database.GetDB()
	if err := db.Create(server).Error; err != nil {
		return nil, fmt.Errorf("falha ao criar servidor no banco: %w", err)
	}

	return server, nil
}
