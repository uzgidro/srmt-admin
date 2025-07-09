package user

type Model struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	PassHash string   `json:"-"`
	Roles    []string `json:"roles"`
}
