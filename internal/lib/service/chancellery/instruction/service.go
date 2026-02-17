package instruction

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	filemodel "srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/instruction"
	instruction_type "srmt-admin/internal/lib/model/instruction-type"
	"srmt-admin/internal/lib/service/fileupload"
)

type RepoInterface interface {
	// Instruction CRUD
	AddInstruction(ctx context.Context, req dto.AddInstructionRequest, createdByID int64) (int64, error)
	EditInstruction(ctx context.Context, id int64, req dto.EditInstructionRequest, updatedByID int64) error
	DeleteInstruction(ctx context.Context, id int64) error
	GetAllInstructions(ctx context.Context, filters dto.GetAllInstructionsFilters) ([]*instruction.ResponseModel, error)
	GetInstructionByID(ctx context.Context, id int64) (*instruction.ResponseModel, error)

	// Status workflow
	GetInstructionStatusHistory(ctx context.Context, instructionID int64) ([]instruction.StatusHistory, error)
	AddInstructionStatusHistoryComment(ctx context.Context, instructionID int64, comment string) error

	// Reference data
	GetAllInstructionTypes(ctx context.Context) ([]instruction_type.Model, error)

	// File linking
	LinkInstructionFiles(ctx context.Context, instructionID int64, fileIDs []int64) error
	UnlinkInstructionFiles(ctx context.Context, instructionID int64) error

	// Document linking
	LinkInstructionDocuments(ctx context.Context, instructionID int64, links []dto.LinkedDocumentRequest, userID int64) error
	UnlinkInstructionDocuments(ctx context.Context, instructionID int64) error

	// File metadata
	AddFile(ctx context.Context, fileData filemodel.Model) (int64, error)
	DeleteFile(ctx context.Context, id int64) error
	GetCategoryByName(ctx context.Context, categoryName string) (fileupload.CategoryModel, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) AddInstruction(ctx context.Context, req dto.AddInstructionRequest, createdByID int64) (int64, error) {
	return s.repo.AddInstruction(ctx, req, createdByID)
}

func (s *Service) EditInstruction(ctx context.Context, id int64, req dto.EditInstructionRequest, updatedByID int64) error {
	return s.repo.EditInstruction(ctx, id, req, updatedByID)
}

func (s *Service) DeleteInstruction(ctx context.Context, id int64) error {
	return s.repo.DeleteInstruction(ctx, id)
}

func (s *Service) GetAllInstructions(ctx context.Context, filters dto.GetAllInstructionsFilters) ([]*instruction.ResponseModel, error) {
	return s.repo.GetAllInstructions(ctx, filters)
}

func (s *Service) GetInstructionByID(ctx context.Context, id int64) (*instruction.ResponseModel, error) {
	return s.repo.GetInstructionByID(ctx, id)
}

func (s *Service) GetInstructionStatusHistory(ctx context.Context, instructionID int64) ([]instruction.StatusHistory, error) {
	return s.repo.GetInstructionStatusHistory(ctx, instructionID)
}

func (s *Service) AddInstructionStatusHistoryComment(ctx context.Context, instructionID int64, comment string) error {
	return s.repo.AddInstructionStatusHistoryComment(ctx, instructionID, comment)
}

func (s *Service) GetAllInstructionTypes(ctx context.Context) ([]instruction_type.Model, error) {
	return s.repo.GetAllInstructionTypes(ctx)
}

func (s *Service) LinkInstructionFiles(ctx context.Context, instructionID int64, fileIDs []int64) error {
	return s.repo.LinkInstructionFiles(ctx, instructionID, fileIDs)
}

func (s *Service) UnlinkInstructionFiles(ctx context.Context, instructionID int64) error {
	return s.repo.UnlinkInstructionFiles(ctx, instructionID)
}

func (s *Service) LinkInstructionDocuments(ctx context.Context, instructionID int64, links []dto.LinkedDocumentRequest, userID int64) error {
	return s.repo.LinkInstructionDocuments(ctx, instructionID, links, userID)
}

func (s *Service) UnlinkInstructionDocuments(ctx context.Context, instructionID int64) error {
	return s.repo.UnlinkInstructionDocuments(ctx, instructionID)
}

func (s *Service) AddFile(ctx context.Context, fileData filemodel.Model) (int64, error) {
	return s.repo.AddFile(ctx, fileData)
}

func (s *Service) DeleteFile(ctx context.Context, id int64) error {
	return s.repo.DeleteFile(ctx, id)
}

func (s *Service) GetCategoryByName(ctx context.Context, categoryName string) (fileupload.CategoryModel, error) {
	return s.repo.GetCategoryByName(ctx, categoryName)
}
