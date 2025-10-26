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
		return nil, fmt.Errorf("failed to create server in database: %w", err)
	}

	return server, nil
}

func (sr *ServerRepository) FindByName(name string) (*model.Server, error) {
	db := database.GetDB()

	server := &model.Server{}
	if err := db.Where("name = ?", name).First(server).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch server by name: %w", err)
	}

	return server, nil
}
