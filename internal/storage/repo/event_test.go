package repo_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/storage"
	repotest "srmt-admin/internal/storage/repo/testing"
)

func TestEventRepository_AddEvent(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully adds event with all required fields", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		req := dto.AddEventRequest{
			Name:                 "Test Meeting",
			Description:          strPtr("Important meeting"),
			Location:             strPtr("Conference Room A"),
			EventDate:            time.Now().Add(24 * time.Hour),
			ResponsibleContactID: fixtures.ContactID,
			EventStatusID:        fixtures.EventStatusID,
			EventTypeID:          fixtures.EventTypeID,
			OrganizationID:       &fixtures.OrgID,
			CreatedByID:          fixtures.UserID,
		}

		eventID, err := repo.AddEvent(ctx, req)

		require.NoError(t, err)
		assert.Greater(t, eventID, int64(0))

		// Verify event was created
		event, err := repo.GetEventByID(ctx, eventID)
		require.NoError(t, err)
		assert.Equal(t, "Test Meeting", event.Name)
		assert.Equal(t, "Important meeting", *event.Description)
		assert.Equal(t, "Conference Room A", *event.Location)
	})

	t.Run("successfully adds event with file links", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		// Create files
		file1Model := file.Model{
			FileName:   "agenda.pdf",
			ObjectKey:  "events/agenda.pdf",
			CategoryID: fixtures.EventsCategoryID,
			CreatedAt:  time.Now(),
		}
		file2Model := file.Model{
			FileName:   "notes.pdf",
			ObjectKey:  "events/notes.pdf",
			CategoryID: fixtures.EventsCategoryID,
			CreatedAt:  time.Now(),
		}
		file1ID, _ := repo.AddFile(ctx, file1Model)
		file2ID, _ := repo.AddFile(ctx, file2Model)

		req := dto.AddEventRequest{
			Name:                 "Event with Files",
			EventDate:            time.Now(),
			ResponsibleContactID: fixtures.ContactID,
			EventStatusID:        fixtures.EventStatusID,
			EventTypeID:          fixtures.EventTypeID,
			CreatedByID:          fixtures.UserID,
			FileIDs:              []int64{file1ID, file2ID},
		}

		eventID, err := repo.AddEvent(ctx, req)

		require.NoError(t, err)

		// Verify files are linked
		event, err := repo.GetEventByID(ctx, eventID)
		require.NoError(t, err)
		assert.Len(t, event.Files, 2)
		assert.Contains(t, []string{event.Files[0].FileName, event.Files[1].FileName}, "agenda.pdf")
		assert.Contains(t, []string{event.Files[0].FileName, event.Files[1].FileName}, "notes.pdf")
	})

	t.Run("returns error on invalid event_type_id", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		req := dto.AddEventRequest{
			Name:                 "Test Event",
			EventDate:            time.Now(),
			ResponsibleContactID: fixtures.ContactID,
			EventStatusID:        fixtures.EventStatusID,
			EventTypeID:          99999, // Invalid
			CreatedByID:          fixtures.UserID,
		}

		_, err := repo.AddEvent(ctx, req)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrForeignKeyViolation)
	})

	t.Run("returns error on invalid event_status_id", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		req := dto.AddEventRequest{
			Name:                 "Test Event",
			EventDate:            time.Now(),
			ResponsibleContactID: fixtures.ContactID,
			EventStatusID:        99999, // Invalid
			EventTypeID:          fixtures.EventTypeID,
			CreatedByID:          fixtures.UserID,
		}

		_, err := repo.AddEvent(ctx, req)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrForeignKeyViolation)
	})

	t.Run("returns error on invalid responsible_contact_id", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		req := dto.AddEventRequest{
			Name:                 "Test Event",
			EventDate:            time.Now(),
			ResponsibleContactID: 99999, // Invalid
			EventStatusID:        fixtures.EventStatusID,
			EventTypeID:          fixtures.EventTypeID,
			CreatedByID:          fixtures.UserID,
		}

		_, err := repo.AddEvent(ctx, req)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrForeignKeyViolation)
	})
}

func TestEventRepository_GetEventByID(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully retrieves event with all relationships", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		req := dto.AddEventRequest{
			Name:                 "Full Event",
			Description:          strPtr("Description"),
			Location:             strPtr("Location"),
			EventDate:            time.Now(),
			ResponsibleContactID: fixtures.ContactID,
			EventStatusID:        fixtures.EventStatusID,
			EventTypeID:          fixtures.EventTypeID,
			OrganizationID:       &fixtures.OrgID,
			CreatedByID:          fixtures.UserID,
		}
		eventID, _ := repo.AddEvent(ctx, req)

		event, err := repo.GetEventByID(ctx, eventID)

		require.NoError(t, err)
		assert.Equal(t, eventID, event.ID)
		assert.Equal(t, "Full Event", event.Name)

		// Check nested objects
		assert.NotNil(t, event.EventStatus)
		assert.NotEmpty(t, event.EventStatus.Name)

		assert.NotNil(t, event.EventType)
		assert.NotEmpty(t, event.EventType.Name)

		assert.NotNil(t, event.ResponsibleContact)
		assert.Equal(t, "Test User", event.ResponsibleContact.FIO)

		assert.NotNil(t, event.Organization)
		assert.Equal(t, "Test Organization", event.Organization.Name)

		assert.NotNil(t, event.CreatedBy)
		assert.Equal(t, "testuser", event.CreatedBy.Login)
	})

	t.Run("successfully retrieves event with files", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		// Create file
		fileModel := file.Model{
			FileName:   "document.pdf",
			ObjectKey:  "events/document.pdf",
			CategoryID: fixtures.EventsCategoryID,
			CreatedAt:  time.Now(),
		}
		fileID, _ := repo.AddFile(ctx, fileModel)

		req := dto.AddEventRequest{
			Name:                 "Event with File",
			EventDate:            time.Now(),
			ResponsibleContactID: fixtures.ContactID,
			EventStatusID:        fixtures.EventStatusID,
			EventTypeID:          fixtures.EventTypeID,
			CreatedByID:          fixtures.UserID,
			FileIDs:              []int64{fileID},
		}
		eventID, _ := repo.AddEvent(ctx, req)

		event, err := repo.GetEventByID(ctx, eventID)

		require.NoError(t, err)
		assert.Len(t, event.Files, 1)
		assert.Equal(t, "document.pdf", event.Files[0].FileName)
	})

	t.Run("returns error for non-existent event", func(t *testing.T) {
		_, err := repo.GetEventByID(ctx, 99999)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}

func TestEventRepository_GetAllEvents(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	fixtures := repotest.LoadFixtures(t, repo)

	// Create test events
	now := time.Now()
	event1Req := dto.AddEventRequest{
		Name:                 "Event 1",
		EventDate:            now.Add(24 * time.Hour),
		ResponsibleContactID: fixtures.ContactID,
		EventStatusID:        3, // Active
		EventTypeID:          1, // Meeting
		CreatedByID:          fixtures.UserID,
	}
	event2Req := dto.AddEventRequest{
		Name:                 "Event 2",
		EventDate:            now.Add(48 * time.Hour),
		ResponsibleContactID: fixtures.ContactID,
		EventStatusID:        1, // Draft
		EventTypeID:          2, // Training
		CreatedByID:          fixtures.UserID,
	}

	event1ID, _ := repo.AddEvent(ctx, event1Req)
	event2ID, _ := repo.AddEvent(ctx, event2Req)

	t.Run("returns all events with no filters", func(t *testing.T) {
		events, err := repo.GetAllEvents(ctx, dto.GetAllEventsFilters{})

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(events), 2)

		// Check our events are included
		ids := make([]int64, len(events))
		for i, e := range events {
			ids[i] = e.ID
		}
		assert.Contains(t, ids, event1ID)
		assert.Contains(t, ids, event2ID)
	})

	t.Run("filters by event_status_id", func(t *testing.T) {
		filters := dto.GetAllEventsFilters{EventStatusIDs: []int{3}} // Active only

		events, err := repo.GetAllEvents(ctx, filters)

		require.NoError(t, err)
		for _, event := range events {
			assert.Equal(t, 3, event.EventStatusID)
		}
	})

	t.Run("filters by multiple event_status_ids", func(t *testing.T) {
		filters := dto.GetAllEventsFilters{EventStatusIDs: []int{1, 3}} // Draft and Active

		events, err := repo.GetAllEvents(ctx, filters)

		require.NoError(t, err)
		// Should include both
		ids := make([]int64, len(events))
		for i, e := range events {
			ids[i] = e.ID
		}
		assert.Contains(t, ids, event1ID)
		assert.Contains(t, ids, event2ID)
	})

	t.Run("filters by event_type_id", func(t *testing.T) {
		filters := dto.GetAllEventsFilters{EventTypeIDs: []int{1}} // Meeting only

		events, err := repo.GetAllEvents(ctx, filters)

		require.NoError(t, err)
		for _, event := range events {
			assert.Equal(t, 1, event.EventTypeID)
		}
	})

	t.Run("filters by date range - start_date only", func(t *testing.T) {
		startDate := now.Add(30 * time.Hour) // Between event1 and event2
		filters := dto.GetAllEventsFilters{StartDate: &startDate}

		events, err := repo.GetAllEvents(ctx, filters)

		require.NoError(t, err)
		// Should only include event2 (48 hours from now)
		for _, event := range events {
			assert.True(t, event.EventDate.After(startDate) || event.EventDate.Equal(startDate))
		}
	})

	t.Run("filters by date range - end_date only", func(t *testing.T) {
		endDate := now.Add(36 * time.Hour) // Between event1 and event2
		filters := dto.GetAllEventsFilters{EndDate: &endDate}

		events, err := repo.GetAllEvents(ctx, filters)

		require.NoError(t, err)
		// Should only include event1 (24 hours from now)
		for _, event := range events {
			assert.True(t, event.EventDate.Before(endDate.AddDate(0, 0, 1)))
		}
	})

	t.Run("filters by date range - both start and end", func(t *testing.T) {
		startDate := now.Add(12 * time.Hour)
		endDate := now.Add(36 * time.Hour)
		filters := dto.GetAllEventsFilters{
			StartDate: &startDate,
			EndDate:   &endDate,
		}

		events, err := repo.GetAllEvents(ctx, filters)

		require.NoError(t, err)
		// Should only include event1
		ids := make([]int64, len(events))
		for i, e := range events {
			ids[i] = e.ID
		}
		assert.Contains(t, ids, event1ID)
		assert.NotContains(t, ids, event2ID)
	})

	t.Run("returns empty array when no matches", func(t *testing.T) {
		filters := dto.GetAllEventsFilters{EventStatusIDs: []int{99}} // Non-existent status

		events, err := repo.GetAllEvents(ctx, filters)

		require.NoError(t, err)
		assert.Equal(t, 0, len(events))
	})
}

func TestEventRepository_EditEvent(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully updates event name", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		// Create event
		createReq := dto.AddEventRequest{
			Name:                 "Original Name",
			EventDate:            time.Now(),
			ResponsibleContactID: fixtures.ContactID,
			EventStatusID:        fixtures.EventStatusID,
			EventTypeID:          fixtures.EventTypeID,
			CreatedByID:          fixtures.UserID,
		}
		eventID, _ := repo.AddEvent(ctx, createReq)

		// Update
		newName := "Updated Name"
		editReq := dto.EditEventRequest{
			Name:        &newName,
			UpdatedByID: fixtures.UserID,
		}

		err := repo.EditEvent(ctx, eventID, editReq)
		require.NoError(t, err)

		// Verify
		event, err := repo.GetEventByID(ctx, eventID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", event.Name)
		assert.NotNil(t, event.UpdatedAt)
		assert.NotNil(t, event.UpdatedBy)
	})

	t.Run("successfully updates event date", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		createReq := dto.AddEventRequest{
			Name:                 "Event",
			EventDate:            time.Now(),
			ResponsibleContactID: fixtures.ContactID,
			EventStatusID:        fixtures.EventStatusID,
			EventTypeID:          fixtures.EventTypeID,
			CreatedByID:          fixtures.UserID,
		}
		eventID, _ := repo.AddEvent(ctx, createReq)

		newDate := time.Now().Add(7 * 24 * time.Hour)
		editReq := dto.EditEventRequest{
			EventDate:   &newDate,
			UpdatedByID: fixtures.UserID,
		}

		err := repo.EditEvent(ctx, eventID, editReq)
		require.NoError(t, err)

		event, err := repo.GetEventByID(ctx, eventID)
		require.NoError(t, err)
		assert.WithinDuration(t, newDate, event.EventDate, time.Second)
	})

	t.Run("successfully replaces file links", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		// Create files
		file1Model := file.Model{FileName: "file1.pdf", ObjectKey: "events/file1.pdf", CategoryID: fixtures.EventsCategoryID, CreatedAt: time.Now()}
		file2Model := file.Model{FileName: "file2.pdf", ObjectKey: "events/file2.pdf", CategoryID: fixtures.EventsCategoryID, CreatedAt: time.Now()}
		file3Model := file.Model{FileName: "file3.pdf", ObjectKey: "events/file3.pdf", CategoryID: fixtures.EventsCategoryID, CreatedAt: time.Now()}

		file1ID, _ := repo.AddFile(ctx, file1Model)
		file2ID, _ := repo.AddFile(ctx, file2Model)
		file3ID, _ := repo.AddFile(ctx, file3Model)

		// Create event with file1 and file2
		createReq := dto.AddEventRequest{
			Name:                 "Event",
			EventDate:            time.Now(),
			ResponsibleContactID: fixtures.ContactID,
			EventStatusID:        fixtures.EventStatusID,
			EventTypeID:          fixtures.EventTypeID,
			CreatedByID:          fixtures.UserID,
			FileIDs:              []int64{file1ID, file2ID},
		}
		eventID, _ := repo.AddEvent(ctx, createReq)

		// Update to file3 only
		editReq := dto.EditEventRequest{
			FileIDs:     []int64{file3ID},
			UpdatedByID: fixtures.UserID,
		}

		err := repo.EditEvent(ctx, eventID, editReq)
		require.NoError(t, err)

		// Verify
		event, err := repo.GetEventByID(ctx, eventID)
		require.NoError(t, err)
		assert.Len(t, event.Files, 1)
		assert.Equal(t, "file3.pdf", event.Files[0].FileName)
	})

	t.Run("returns error for non-existent event", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		newName := "Test"
		editReq := dto.EditEventRequest{
			Name:        &newName,
			UpdatedByID: fixtures.UserID,
		}

		err := repo.EditEvent(ctx, 99999, editReq)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})

	t.Run("returns error on invalid event_type_id", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		createReq := dto.AddEventRequest{
			Name:                 "Event",
			EventDate:            time.Now(),
			ResponsibleContactID: fixtures.ContactID,
			EventStatusID:        fixtures.EventStatusID,
			EventTypeID:          fixtures.EventTypeID,
			CreatedByID:          fixtures.UserID,
		}
		eventID, _ := repo.AddEvent(ctx, createReq)

		invalidTypeID := 99999
		editReq := dto.EditEventRequest{
			EventTypeID: &invalidTypeID,
			UpdatedByID: fixtures.UserID,
		}

		err := repo.EditEvent(ctx, eventID, editReq)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrForeignKeyViolation)
	})
}

func TestEventRepository_DeleteEvent(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("successfully deletes event", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		req := dto.AddEventRequest{
			Name:                 "To Delete",
			EventDate:            time.Now(),
			ResponsibleContactID: fixtures.ContactID,
			EventStatusID:        fixtures.EventStatusID,
			EventTypeID:          fixtures.EventTypeID,
			CreatedByID:          fixtures.UserID,
		}
		eventID, _ := repo.AddEvent(ctx, req)

		err := repo.DeleteEvent(ctx, eventID)
		require.NoError(t, err)

		// Verify deletion
		_, err = repo.GetEventByID(ctx, eventID)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})

	t.Run("successfully deletes event with file links (cascade)", func(t *testing.T) {
		fixtures := repotest.LoadFixtures(t, repo)

		// Create file
		fileModel := file.Model{
			FileName:   "file.pdf",
			ObjectKey:  "events/file.pdf",
			CategoryID: fixtures.EventsCategoryID,
			CreatedAt:  time.Now(),
		}
		fileID, _ := repo.AddFile(ctx, fileModel)

		req := dto.AddEventRequest{
			Name:                 "Event with File",
			EventDate:            time.Now(),
			ResponsibleContactID: fixtures.ContactID,
			EventStatusID:        fixtures.EventStatusID,
			EventTypeID:          fixtures.EventTypeID,
			CreatedByID:          fixtures.UserID,
			FileIDs:              []int64{fileID},
		}
		eventID, _ := repo.AddEvent(ctx, req)

		err := repo.DeleteEvent(ctx, eventID)
		require.NoError(t, err)

		// Verify event is deleted
		_, err = repo.GetEventByID(ctx, eventID)
		assert.ErrorIs(t, err, storage.ErrNotFound)

		// Verify file still exists (only link is deleted)
		_, err = repo.GetFileByID(ctx, fileID)
		assert.NoError(t, err)
	})

	t.Run("returns error for non-existent event", func(t *testing.T) {
		err := repo.DeleteEvent(ctx, 99999)

		require.Error(t, err)
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
}

func TestEventRepository_GetEventStatuses(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("returns all event statuses", func(t *testing.T) {
		statuses, err := repo.GetEventStatuses(ctx)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(statuses), 6) // Draft, Planned, Active, Completed, Cancelled, Postponed

		// Verify key statuses exist
		names := make([]string, len(statuses))
		for i, s := range statuses {
			names[i] = s.Name
		}
		assert.Contains(t, names, "Draft")
		assert.Contains(t, names, "Active")
		assert.Contains(t, names, "Completed")
	})
}

func TestEventRepository_GetEventTypes(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	t.Run("returns all event types", func(t *testing.T) {
		types, err := repo.GetEventTypes(ctx)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(types), 4) // Meeting, Training, Inspection, Maintenance

		// Verify key types exist
		names := make([]string, len(types))
		for i, typ := range types {
			names[i] = typ.Name
		}
		assert.Contains(t, names, "Meeting")
		assert.Contains(t, names, "Training")
		assert.Contains(t, names, "Inspection")
		assert.Contains(t, names, "Maintenance")
	})
}

func TestEventRepository_LinkUnlinkEventFiles(t *testing.T) {
	testDB := repotest.SetupTestDB(t)
	defer testDB.Cleanup(t)

	repo := testDB.NewRepo()
	ctx := context.Background()

	fixtures := repotest.LoadFixtures(t, repo)

	t.Run("successfully links multiple files", func(t *testing.T) {
		// Create event
		req := dto.AddEventRequest{
			Name:                 "Event",
			EventDate:            time.Now(),
			ResponsibleContactID: fixtures.ContactID,
			EventStatusID:        fixtures.EventStatusID,
			EventTypeID:          fixtures.EventTypeID,
			CreatedByID:          fixtures.UserID,
		}
		eventID, _ := repo.AddEvent(ctx, req)

		// Create files
		file1Model := file.Model{FileName: "file1.pdf", ObjectKey: "events/file1.pdf", CategoryID: fixtures.EventsCategoryID, CreatedAt: time.Now()}
		file2Model := file.Model{FileName: "file2.pdf", ObjectKey: "events/file2.pdf", CategoryID: fixtures.EventsCategoryID, CreatedAt: time.Now()}
		file1ID, _ := repo.AddFile(ctx, file1Model)
		file2ID, _ := repo.AddFile(ctx, file2Model)

		// Link files
		err := repo.LinkEventFiles(ctx, eventID, []int64{file1ID, file2ID})
		require.NoError(t, err)

		// Verify
		event, err := repo.GetEventByID(ctx, eventID)
		require.NoError(t, err)
		assert.Len(t, event.Files, 2)
	})

	t.Run("successfully unlinks all files", func(t *testing.T) {
		// Create event with files
		fileModel := file.Model{FileName: "file.pdf", ObjectKey: "events/unique-file.pdf", CategoryID: fixtures.EventsCategoryID, CreatedAt: time.Now()}
		fileID, _ := repo.AddFile(ctx, fileModel)

		req := dto.AddEventRequest{
			Name:                 "Event",
			EventDate:            time.Now(),
			ResponsibleContactID: fixtures.ContactID,
			EventStatusID:        fixtures.EventStatusID,
			EventTypeID:          fixtures.EventTypeID,
			CreatedByID:          fixtures.UserID,
			FileIDs:              []int64{fileID},
		}
		eventID, _ := repo.AddEvent(ctx, req)

		// Unlink files
		err := repo.UnlinkEventFiles(ctx, eventID)
		require.NoError(t, err)

		// Verify
		event, err := repo.GetEventByID(ctx, eventID)
		require.NoError(t, err)
		assert.Len(t, event.Files, 0)
	})

	t.Run("handles duplicate file links gracefully", func(t *testing.T) {
		fileModel := file.Model{FileName: "dup.pdf", ObjectKey: "events/dup.pdf", CategoryID: fixtures.EventsCategoryID, CreatedAt: time.Now()}
		fileID, _ := repo.AddFile(ctx, fileModel)

		req := dto.AddEventRequest{
			Name:                 "Event",
			EventDate:            time.Now(),
			ResponsibleContactID: fixtures.ContactID,
			EventStatusID:        fixtures.EventStatusID,
			EventTypeID:          fixtures.EventTypeID,
			CreatedByID:          fixtures.UserID,
			FileIDs:              []int64{fileID},
		}
		eventID, _ := repo.AddEvent(ctx, req)

		// Try to link same file again
		err := repo.LinkEventFiles(ctx, eventID, []int64{fileID})
		// Should not error (ON CONFLICT DO NOTHING)
		require.NoError(t, err)

		// Should still have only one link
		event, err := repo.GetEventByID(ctx, eventID)
		require.NoError(t, err)
		assert.Len(t, event.Files, 1)
	})
}
