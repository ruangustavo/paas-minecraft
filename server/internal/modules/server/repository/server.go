package repository

import (
	"fmt"
	"paas-minecraft/internal/modules/server/model"
	"paas-minecraft/internal/shared/database"

	"github.com/google/uuid"
)

type ServerRepository struct{}

func NewServerRepository() *ServerRepository {
	return &ServerRepository{}
}

func (sr *ServerRepository) Create(name string, port int, subdomain string, userID uuid.UUID) (*model.Server, error) {
	server := &model.Server{
		Name:      name,
		Port:      port,
		Subdomain: subdomain,
		Status:    "running",
		UserID:    userID,
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

func (sr *ServerRepository) FindByID(id uuid.UUID) (*model.Server, error) {
	db := database.GetDB()

	server := &model.Server{}
	if err := db.Where("id = ?", id).First(server).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch server by id: %w", err)
	}

	return server, nil
}

func (sr *ServerRepository) FindAll() ([]*model.Server, error) {
	db := database.GetDB()

	var servers []*model.Server
	if err := db.Order("created_at desc").Find(&servers).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch all servers: %w", err)
	}

	return servers, nil
}

func (sr *ServerRepository) FindAllByUserID(userID uuid.UUID) ([]*model.Server, error) {
	db := database.GetDB()

	var servers []*model.Server
	if err := db.Where("user_id = ?", userID).Order("created_at desc").Find(&servers).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch servers by user ID: %w", err)
	}

	return servers, nil
}

func (sr *ServerRepository) FindByIDAndUserID(id uuid.UUID, userID uuid.UUID) (*model.Server, error) {
	db := database.GetDB()

	server := &model.Server{}
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(server).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch server by id and user: %w", err)
	}

	return server, nil
}

func (sr *ServerRepository) UpdateStatus(id uuid.UUID, status string) error {
	db := database.GetDB()

	if err := db.Model(&model.Server{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to update server status: %w", err)
	}

	return nil
}

func (sr *ServerRepository) Delete(id uuid.UUID) error {
	db := database.GetDB()

	if err := db.Where("id = ?", id).Delete(&model.Server{}).Error; err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	return nil
}
