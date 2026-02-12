package dashboard

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/model/hrm/dashboard"
	"sync"
)

type RepoInterface interface {
	GetHRMDashboardWidgets(ctx context.Context) (*dashboard.Widgets, error)
	GetHRMDashboardTasks(ctx context.Context, userID int64) ([]dashboard.Task, error)
	GetHRMDashboardEvents(ctx context.Context) ([]dashboard.Event, error)
	GetHRMNotifications(ctx context.Context, userID int64) ([]*dashboard.Notification, error)
	GetHRMDashboardActivity(ctx context.Context) ([]dashboard.Activity, error)
	GetHRMUpcomingBirthdays(ctx context.Context) ([]dashboard.Birthday, error)
	GetHRMProbationEmployees(ctx context.Context) ([]dashboard.ProbationEmployee, error)
	MarkHRMNotificationRead(ctx context.Context, notificationID int64, userID int64) error
	MarkAllHRMNotificationsRead(ctx context.Context, userID int64) error
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) GetDashboard(ctx context.Context, userID int64) (*dashboard.Data, error) {
	data := &dashboard.Data{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	errs := make([]error, 0)

	collect := func(fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	collect(func() {
		w, err := s.repo.GetHRMDashboardWidgets(ctx)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errs = append(errs, err)
			s.log.Error("failed to get dashboard widgets", "error", err)
			return
		}
		data.Widgets = *w
	})

	collect(func() {
		tasks, err := s.repo.GetHRMDashboardTasks(ctx, userID)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errs = append(errs, err)
			s.log.Error("failed to get dashboard tasks", "error", err)
			return
		}
		data.Tasks = tasks
	})

	collect(func() {
		events, err := s.repo.GetHRMDashboardEvents(ctx)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errs = append(errs, err)
			s.log.Error("failed to get dashboard events", "error", err)
			return
		}
		data.Events = events
	})

	collect(func() {
		notifications, err := s.repo.GetHRMNotifications(ctx, userID)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errs = append(errs, err)
			s.log.Error("failed to get dashboard notifications", "error", err)
			return
		}
		notifs := make([]dashboard.Notification, len(notifications))
		for i, n := range notifications {
			notifs[i] = *n
		}
		data.Notifications = notifs
	})

	collect(func() {
		activity, err := s.repo.GetHRMDashboardActivity(ctx)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errs = append(errs, err)
			s.log.Error("failed to get dashboard activity", "error", err)
			return
		}
		data.RecentActivity = activity
	})

	collect(func() {
		birthdays, err := s.repo.GetHRMUpcomingBirthdays(ctx)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errs = append(errs, err)
			s.log.Error("failed to get upcoming birthdays", "error", err)
			return
		}
		data.UpcomingBirthdays = birthdays
	})

	collect(func() {
		employees, err := s.repo.GetHRMProbationEmployees(ctx)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errs = append(errs, err)
			s.log.Error("failed to get probation employees", "error", err)
			return
		}
		data.ProbationEmployees = employees
	})

	wg.Wait()

	// Ensure nil slices become empty arrays in JSON
	if data.Tasks == nil {
		data.Tasks = []dashboard.Task{}
	}
	if data.Events == nil {
		data.Events = []dashboard.Event{}
	}
	if data.Notifications == nil {
		data.Notifications = []dashboard.Notification{}
	}
	if data.RecentActivity == nil {
		data.RecentActivity = []dashboard.Activity{}
	}
	if data.UpcomingBirthdays == nil {
		data.UpcomingBirthdays = []dashboard.Birthday{}
	}
	if data.ProbationEmployees == nil {
		data.ProbationEmployees = []dashboard.ProbationEmployee{}
	}

	return data, nil
}

func (s *Service) MarkNotificationRead(ctx context.Context, notificationID int64, userID int64) error {
	return s.repo.MarkHRMNotificationRead(ctx, notificationID, userID)
}

func (s *Service) MarkAllNotificationsRead(ctx context.Context, userID int64) error {
	return s.repo.MarkAllHRMNotificationsRead(ctx, userID)
}
