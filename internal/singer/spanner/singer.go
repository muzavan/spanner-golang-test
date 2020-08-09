package spanner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/muzavan/spanner-golang-test/internal/singer"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"

	"cloud.google.com/go/civil"
	googleSpanner "cloud.google.com/go/spanner"
)

var singerTableName = "Singers"
var singerColumns = []string{
	"SingerID",
	"FirstName",
	"LastName",
	"SingerInfo",
	"BirthDate",
}

// SingerRow represent the row in Singer table
type SingerRow struct {
	SingerID   int64
	FirstName  googleSpanner.NullString
	LastName   googleSpanner.NullString
	SingerInfo []byte
	BirthDate  googleSpanner.NullDate
}

func (singerRow *SingerRow) toDetail() (*singer.Detail, error) {
	detail := &singer.Detail{
		SingerID: singerRow.SingerID,
	}

	if !singerRow.BirthDate.IsNull() {
		detail.BirthDate = singerRow.BirthDate.Date.In(time.Local)
	}

	if !singerRow.FirstName.IsNull() {
		detail.FirstName = singerRow.FirstName.String()
	}

	if !singerRow.LastName.IsNull() {
		detail.LastName = singerRow.LastName.String()
	}

	var detailInfo singer.Info
	err := json.Unmarshal(singerRow.SingerInfo, &detailInfo)

	if err != nil {
		return nil, err
	}
	detail.Info = detailInfo

	return detail, nil
}

// SingerProvider implements singer.Repository
type SingerProvider struct {
	Client *googleSpanner.Client
}

// Create store singer to Spanner
func (repo *SingerProvider) Create(ctx context.Context, payload singer.CreatePayload) error {
	newSinger := SingerRow{
		SingerID: payload.SingerID,
		FirstName: googleSpanner.NullString{
			Valid:     len(payload.FirstName) > 0,
			StringVal: payload.FirstName,
		},
		LastName: googleSpanner.NullString{
			Valid:     len(payload.LastName) > 0,
			StringVal: payload.LastName,
		},
		BirthDate: googleSpanner.NullDate{
			Valid: !payload.BirthDate.IsZero(),
			Date:  civil.DateOf(payload.BirthDate),
		},
	}

	infoByte, err := json.Marshal(payload.Info)
	if err != nil {
		return fmt.Errorf("%w - %s", singer.ErrBadValue, err.Error())
	}

	newSinger.SingerInfo = infoByte

	newSingerMutation, err := googleSpanner.InsertStruct(singerTableName, newSinger)

	if err != nil {
		return fmt.Errorf("%w - %s", singer.ErrUnknown, err.Error())
	}

	_, err = repo.Client.Apply(ctx, []*googleSpanner.Mutation{newSingerMutation})

	if err != nil {
		code := googleSpanner.ErrCode(err)
		if code == codes.AlreadyExists {
			return fmt.Errorf("%w - %s", singer.ErrDuplicate, err.Error())
		}
		return fmt.Errorf("%w - %s", singer.ErrUnknown, err.Error())
	}
	return nil
}

// List fetch singers from Spanner based on filter condition
func (repo *SingerProvider) List(ctx context.Context, filter singer.FilterPayload) ([]*singer.Detail, error) {
	sqlQuery := fmt.Sprintf("SELECT %s FROM %s", strings.Join(singerColumns, ", "), singerTableName)

	whereClause := []string{}
	if len(filter.Name) > 0 {
		whereClause = append(whereClause, "( FirstName LIKE @name OR LastName LIKE @name )")
	}

	if !filter.BirthDateStart.IsZero() {
		whereClause = append(whereClause, "BirthDate >= @birthDateStart")
	}

	if !filter.BirthDateEnd.IsZero() {
		whereClause = append(whereClause, "BirthDate <= @birthDateEnd")
	}

	if len(whereClause) > 0 {
		sqlWhereClause := strings.Join(whereClause, " AND ")
		sqlQuery = fmt.Sprintf("%s WHERE %s", sqlQuery, sqlWhereClause)
	}
	stmt := googleSpanner.Statement{
		SQL: sqlQuery,
		Params: map[string]interface{}{
			"name":           fmt.Sprintf("%%%s%%", filter.Name),
			"birthDateStart": civil.DateOf(filter.BirthDateStart),
			"birthDateEnd":   civil.DateOf(filter.BirthDateEnd),
		},
	}

	iter := repo.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	details := []*singer.Detail{}

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("%w - %s", singer.ErrUnknown, err.Error())
		}

		var singerRow SingerRow
		err = row.ToStruct(&singerRow)

		if err != nil {
			return nil, fmt.Errorf("%w - %s", singer.ErrBadValue, err.Error())
		}

		detail, err := singerRow.toDetail()

		if err != nil {
			return nil, fmt.Errorf("%w - %s", singer.ErrBadValue, err.Error())
		}

		details = append(details, detail)
	}

	return details, nil
}

// Get fetch singer from Spanner based on its ID
func (repo *SingerProvider) Get(ctx context.Context, singerID int64) (*singer.Detail, error) {
	row, err := repo.Client.Single().ReadRow(ctx, singerTableName, googleSpanner.Key{singerID}, singerColumns)

	if err != nil {
		code := googleSpanner.ErrCode(err)
		if code == codes.NotFound {
			return nil, fmt.Errorf("%w - %s", singer.ErrNotFound, err.Error())
		}

		return nil, fmt.Errorf("%w - %s", singer.ErrUnknown, err.Error())
	}

	var singerRow SingerRow
	err = row.ToStruct(&singerRow)

	if err != nil {
		return nil, fmt.Errorf("%w - %s", singer.ErrBadValue, err.Error())
	}

	return singerRow.toDetail()
}
