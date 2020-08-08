package singer

import "time"

// Repository reads/writes the data to the actual storage
type Repository interface {
	Create(context.Context, CreatePayload) error
	List(context.Context, FilterPayload) ([]Detail, error)
	Get(context.Context, string) (*Detail, error)
}

// CreatePayload represents the necessary field for storing singer data
type CreatePayload struct {
	SingerID string
	FirstName string
	LastName string
	Info SingerInfo
	BirthDate time.Time
}

// FilterPayload defines the possible filter condition 
type FilterPayload struct {
	Name string
	BirthDateStart time.Time
	BirthDateEnd time.Time
}

type Info struct {
	Songs []string
	Awards []string
}

type Detail struct {
	SingerID string
	FirstName string
	LastName string
	Info Info
	BirthDate time.Time
}