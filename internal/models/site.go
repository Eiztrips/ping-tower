package models

import (
	"time"
)

type Site struct {
	ID          int       `json:"id"`
	URL         string    `json:"url"`
	Status      string    `json:"status"`
	LastChecked time.Time `json:"last_checked"`
}