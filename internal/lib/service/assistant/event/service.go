package event

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/contact"
	"srmt-admin/internal/lib/model/event"
	"srmt-admin/internal/lib/model/event_status"
	"srmt-admin/internal/lib/model/event_type"
	filemodel "srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/service/fileupload"
)

type RepoInterface interface {
	// Event CRUD
	AddEvent(ctx context.Context, req dto.AddEventRequest) (int64, error)
	EditEvent(ctx context.Context, eventID int64, req dto.EditEventRequest) error
	DeleteEvent(ctx context.Context, id int64) error
	GetAllEvents(ctx context.Context, filters dto.GetAllEventsFilters) ([]*event.Model, error)
	GetAllEventsShort(ctx context.Context, filters dto.GetAllEventsFilters) ([]*event.Model, error)
	GetEventByID(ctx context.Context, id int64) (*event.Model, error)

	// Reference data
	GetEventStatuses(ctx context.Context) ([]event_status.Model, error)
	GetEventTypes(ctx context.Context) ([]event_type.Model, error)

	// Contact operations (for inline contact creation)
	AddContact(ctx context.Context, req dto.AddContactRequest) (int64, error)
	GetContactByID(ctx context.Context, id int64) (*contact.Model, error)

	// File linking
	LinkEventFiles(ctx context.Context, eventID int64, fileIDs []int64) error
	UnlinkEventFiles(ctx context.Context, eventID int64) error

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

func (s *Service) AddEvent(ctx context.Context, req dto.AddEventRequest) (int64, error) {
	return s.repo.AddEvent(ctx, req)
}

func (s *Service) EditEvent(ctx context.Context, eventID int64, req dto.EditEventRequest) error {
	return s.repo.EditEvent(ctx, eventID, req)
}

func (s *Service) DeleteEvent(ctx context.Context, id int64) error {
	return s.repo.DeleteEvent(ctx, id)
}

func (s *Service) GetAllEvents(ctx context.Context, filters dto.GetAllEventsFilters) ([]*event.Model, error) {
	return s.repo.GetAllEvents(ctx, filters)
}

func (s *Service) GetAllEventsShort(ctx context.Context, filters dto.GetAllEventsFilters) ([]*event.Model, error) {
	return s.repo.GetAllEventsShort(ctx, filters)
}

func (s *Service) GetEventByID(ctx context.Context, id int64) (*event.Model, error) {
	return s.repo.GetEventByID(ctx, id)
}

func (s *Service) GetEventStatuses(ctx context.Context) ([]event_status.Model, error) {
	return s.repo.GetEventStatuses(ctx)
}

func (s *Service) GetEventTypes(ctx context.Context) ([]event_type.Model, error) {
	return s.repo.GetEventTypes(ctx)
}

func (s *Service) AddContact(ctx context.Context, req dto.AddContactRequest) (int64, error) {
	return s.repo.AddContact(ctx, req)
}

func (s *Service) GetContactByID(ctx context.Context, id int64) (*contact.Model, error) {
	return s.repo.GetContactByID(ctx, id)
}

func (s *Service) LinkEventFiles(ctx context.Context, eventID int64, fileIDs []int64) error {
	return s.repo.LinkEventFiles(ctx, eventID, fileIDs)
}

func (s *Service) UnlinkEventFiles(ctx context.Context, eventID int64) error {
	return s.repo.UnlinkEventFiles(ctx, eventID)
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
