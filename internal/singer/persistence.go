package singer

import (
	"context"
	"time"
)

// Repository reads/writes the data to the actual storage
type Repository interface {
	Create(context.Context, CreatePayload) error
	List(context.Context, FilterPayload) ([]*Detail, error)
	Get(context.Context, int64) (*Detail, error)
}

// CreatePayload represents the necessary field for storing singer data
type CreatePayload struct {
	SingerID  int64
	FirstName string
	LastName  string
	Info      Info
	BirthDate time.Time
}

// FilterPayload defines the possible filter condition
type FilterPayload struct {
	Name           string
	BirthDateStart time.Time
	BirthDateEnd   time.Time
}

// Info contains additional detail for Singer
type Info struct {
	Songs  []string
	Awards []string
}

// Detail contains main information for Singer
type Detail struct {
	SingerID  int64
	FirstName string
	LastName  string
	Info      Info
	BirthDate time.Time
}
