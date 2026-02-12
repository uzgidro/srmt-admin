package dashboard

type Data struct {
	Widgets            Widgets             `json:"widgets"`
	Tasks              []Task              `json:"tasks"`
	Events             []Event             `json:"events"`
	Notifications      []Notification      `json:"notifications"`
	RecentActivity     []Activity          `json:"recent_activity"`
	UpcomingBirthdays  []Birthday          `json:"upcoming_birthdays"`
	ProbationEmployees []ProbationEmployee `json:"probation_employees"`
}

type Widgets struct {
	TotalEmployees       int     `json:"total_employees"`
	OnVacation           int     `json:"on_vacation"`
	OnSickLeave          int     `json:"on_sick_leave"`
	OpenVacancies        int     `json:"open_vacancies"`
	PendingApprovals     int     `json:"pending_approvals"`
	NewEmployeesMonth    int     `json:"new_employees_month"`
	DismissedMonth       int     `json:"dismissed_month"`
	AvgAttendancePercent float64 `json:"avg_attendance_percent"`
}

type Task struct {
	ID              int64   `json:"id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Type            string  `json:"type"`
	Priority        string  `json:"priority"`
	DueDate         string  `json:"due_date"`
	RelatedEntity   *string `json:"related_entity,omitempty"`
	RelatedEntityID *int64  `json:"related_entity_id,omitempty"`
	Assignee        *string `json:"assignee,omitempty"`
}

type Event struct {
	ID                int64   `json:"id"`
	Title             string  `json:"title"`
	Type              string  `json:"type"`
	Date              string  `json:"date"`
	Time              *string `json:"time,omitempty"`
	Location          *string `json:"location,omitempty"`
	ParticipantsCount *int    `json:"participants_count,omitempty"`
}

type Notification struct {
	ID        int64   `json:"id"`
	Title     string  `json:"title"`
	Message   string  `json:"message"`
	Type      string  `json:"type"`
	Read      bool    `json:"read"`
	CreatedAt string  `json:"created_at"`
	Link      *string `json:"link,omitempty"`
}

type Activity struct {
	ID           int64   `json:"id"`
	Type         string  `json:"type"`
	Description  string  `json:"description"`
	EmployeeName *string `json:"employee_name,omitempty"`
	Timestamp    string  `json:"timestamp"`
}

type Birthday struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Position   string  `json:"position"`
	Department string  `json:"department"`
	Date       string  `json:"date"`
	Avatar     *string `json:"avatar,omitempty"`
}

type ProbationEmployee struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Position   string  `json:"position"`
	Department string  `json:"department"`
	StartDate  string  `json:"start_date"`
	EndDate    string  `json:"end_date"`
	Progress   int     `json:"progress"`
	Status     string  `json:"status"`
	Mentor     *string `json:"mentor,omitempty"`
}
