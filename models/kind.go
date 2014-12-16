package models

import "time"

type Kind struct {
	Primary     int       `db:"primary"`
	ID          string    `db:"id"`
	Description string    `db:"description"`
	Critical    bool      `db:"critical"`
	ClientID    string    `db:"client_id"`
	CreatedAt   time.Time `db:"created_at"`
	Template    string    `db:"template_id"`
}

func (k Kind) TemplateToUse() string {
	if k.Template != "" {
		return k.Template
	}

	return DefaultTemplateID
}
