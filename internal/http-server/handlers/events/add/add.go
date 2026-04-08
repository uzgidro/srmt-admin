package add

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/contact"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

// Request (JSON DTO)
type addRequest struct {
	Name                 string    `json:"name" validate:"required"`
	Description          *string   `json:"description,omitempty"`
	Location             *string   `json:"location,omitempty"`
	EventDate            time.Time `json:"event_date" validate:"required"`
	ResponsibleContactID *int64    `json:"responsible_contact_id,omitempty"`
	EventTypeID          int       `json:"event_type_id" validate:"required"`
	OrganizationID       *int64    `json:"organization_id,omitempty"`

	// Fields for creating a new contact if not using existing ID
	ResponsibleFIO   *string `json:"responsible_fio,omitempty"`
	ResponsiblePhone *string `json:"responsible_phone,omitempty"`

	FileIDs []int64 `json:"file_ids,omitempty"`
}

type addResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

type eventAdder interface {
	AddEvent(ctx context.Context, req dto.AddEventRequest) (int64, error)
	AddContact(ctx context.Context, req dto.AddContactRequest) (int64, error)
	GetContactByID(ctx context.Context, id int64) (*contact.Model, error)
}

func New(log *slog.Logger, adder eventAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.event.add.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req addRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		// Validate request
		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		// Handle responsible contact - either use existing ID or create new
		var responsibleContactID int64

		if req.ResponsibleContactID != nil {
			// Use existing contact
			contactID := *req.ResponsibleContactID

			// Verify contact exists
			_, err = adder.GetContactByID(r.Context(), contactID)
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Contact with this ID does not exist"))
					return
				}
				log.Error("failed to verify contact", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to verify contact"))
				return
			}

			responsibleContactID = contactID
		} else {
			// Create new contact from name and phone
			if req.ResponsibleFIO == nil || req.ResponsiblePhone == nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Either 'responsible_contact_id' or both 'responsible_fio' and 'responsible_phone' are required"))
				return
			}

			// Create contact
			contactReq := dto.AddContactRequest{
				Name:  *req.ResponsibleFIO,
				Phone: req.ResponsiblePhone,
			}

			contactID, err := adder.AddContact(r.Context(), contactReq)
			if err != nil {
				if errors.Is(err, storage.ErrDuplicate) {
					log.Warn("duplicate contact phone", "phone", *req.ResponsiblePhone)
					render.Status(r, http.StatusConflict)
					render.JSON(w, r, resp.BadRequest("Contact with this phone already exists"))
					return
				}
				log.Error("failed to create contact", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to create contact"))
				return
			}

			responsibleContactID = contactID
			log.Info("created new contact", slog.Int64("contact_id", contactID))
		}

		// Get default "Active" status (ID = 3)
		eventStatusID := 3 // Active status

		// Create event
		eventReq := dto.AddEventRequest{
			Name:                 req.Name,
			Description:          req.Description,
			Location:             req.Location,
			EventDate:            req.EventDate,
			ResponsibleContactID: responsibleContactID,
			EventStatusID:        eventStatusID,
			EventTypeID:          req.EventTypeID,
			OrganizationID:       req.OrganizationID,
			CreatedByID:          userID,
			FileIDs:              req.FileIDs,
		}

		id, err := adder.AddEvent(r.Context(), eventReq)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation", "event_type_id", req.EventTypeID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid event_type_id, organization_id, or contact_id"))
				return
			}
			log.Error("failed to add event", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add event"))
			return
		}

		log.Info("event added successfully",
			slog.Int64("id", id),
			slog.Int("total_files", len(req.FileIDs)),
		)

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, addResponse{
			Response: resp.Created(),
			ID:       id,
		})
	}
}
