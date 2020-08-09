package spanner

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/muzavan/spanner-golang-test/internal/singer"

	googleSpanner "cloud.google.com/go/spanner"

	adminapi "cloud.google.com/go/spanner/admin/database/apiv1"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/api/option"
	adminpb "google.golang.org/genproto/googleapis/spanner/admin/database/v1"
	instancepb "google.golang.org/genproto/googleapis/spanner/admin/instance/v1"
	"google.golang.org/grpc"
)

const spannerProjectName = "test-project"
const spannerInstanceName = "test-instance"
const spannerDbName = "MusicDb"

type SingerTestSuite struct {
	suite.Suite
	provider       *SingerProvider
	grpcConn       *grpc.ClientConn
	dbClient       *adminapi.DatabaseAdminClient
	instanceClient *instance.InstanceAdminClient
}

func TestSingerTestSuite(t *testing.T) {
	suite.Run(t, new(SingerTestSuite))
}

// SetupSuite will create the database instance, and later on will be dropped, as suggested by: https://github.com/GoogleCloudPlatform/cloud-spanner-emulator
func (s *SingerTestSuite) SetupSuite() {
	if _, ok := os.LookupEnv("SPANNER_EMULATOR_HOST"); !ok {
		// Will be used by SpannerClient, it checks the env var
		os.Setenv("SPANNER_EMULATOR_HOST", "localhost:9010")

		defer os.Unsetenv("SPANNER_EMULATOR_HOST")
	}

	spannerServerAddress := os.Getenv("SPANNER_EMULATOR_HOST")

	ctx := context.Background()

	dialCtx := ctx

	conn, err := grpc.DialContext(dialCtx, spannerServerAddress, grpc.WithInsecure())

	if err != nil {
		s.FailNow("Can't create grpc conn for database admin client", err)
	}

	// Create Instance
	instanceClient, err := instance.NewInstanceAdminClient(ctx, option.WithGRPCConn(conn))
	if err != nil {
		s.FailNow("Can't create instance client", err)
	}

	createInstanceReq := &instancepb.CreateInstanceRequest{
		Parent:     fmt.Sprintf("projects/%s", spannerProjectName),
		InstanceId: spannerInstanceName,
	}

	op, err := instanceClient.CreateInstance(ctx, createInstanceReq)
	if err != nil {
		s.FailNow("Can't create instance operation", err)
	}

	_, err = op.Wait(ctx)
	if err != nil {
		s.FailNow("Can't finish create instance operation")
	}

	// Create DB and Table
	client, err := adminapi.NewDatabaseAdminClient(ctx, option.WithoutAuthentication(), option.WithGRPCConn(conn))

	if err != nil {
		s.FailNow("Can't create database admin client", err)
	}

	// Using same table as https://cloud.google.com/spanner/docs/quickstart-console
	def, err := ioutil.ReadFile("../../ddl/spanner/001_create_singer_table.sql")

	if err != nil {
		s.FailNow("Can't read the ddl file", err)
	}

	extras := []string{}
	for _, stmt := range strings.Split(string(def), ";") {
		trimmedStmt := strings.TrimSpace(stmt)
		if len(trimmedStmt) > 0 {
			extras = append(extras, trimmedStmt)
		}
	}

	_, err = client.CreateDatabase(ctx, &adminpb.CreateDatabaseRequest{
		Parent:          fmt.Sprintf("projects/%s/instances/%s", spannerProjectName, spannerInstanceName),
		CreateStatement: fmt.Sprintf("CREATE DATABASE %s", spannerDbName),
		ExtraStatements: extras,
	})

	if err != nil {
		s.FailNow("Can't create the database", err)
	}

	spannerClient, err := googleSpanner.NewClient(ctx,
		fmt.Sprintf("projects/%s/instances/%s/databases/%s", spannerProjectName, spannerInstanceName, spannerDbName),
		option.WithoutAuthentication(),
		option.WithGRPCConn(conn),
	)

	if err != nil {
		s.FailNow("Can't create the database client", err)
	}

	s.provider = &SingerProvider{
		Client: spannerClient,
	}
	s.grpcConn = conn
	s.dbClient = client
	s.instanceClient = instanceClient
}

func (s *SingerTestSuite) TearDownSuite() {
	ctx := context.Background()
	err := s.dbClient.DropDatabase(ctx, &adminpb.DropDatabaseRequest{
		Database: fmt.Sprintf("projects/%s/instances/%s/databases/%s", spannerProjectName, spannerInstanceName, spannerDbName),
	})

	if err != nil {
		s.FailNowf("Can't drop spanner db", "%w", err)
	}

	deleteReq := &instancepb.DeleteInstanceRequest{
		Name: fmt.Sprintf("projects/%s/instances/%s", spannerProjectName, spannerInstanceName),
	}
	err = s.instanceClient.DeleteInstance(ctx, deleteReq)
	if err != nil {
		s.FailNowf("Can't drop spanner instance", "%w", err)
	}

	s.provider.Client.Close()
	s.dbClient.Close()
	s.instanceClient.Close()
	s.grpcConn.Close()
}

func (s *SingerTestSuite) SetupTest() {
	ctx := context.Background()

	// TRUNCATE Singers Related Table
	_, err := s.provider.Client.ReadWriteTransaction(ctx, func(ctx context.Context, tx *googleSpanner.ReadWriteTransaction) error {
		stmt := googleSpanner.Statement{
			SQL: fmt.Sprintf("DELETE FROM %s WHERE 1 = 1", singerTableName),
		}

		_, err := tx.Update(ctx, stmt)
		return err
	})

	if err != nil {
		s.T().Log("Can't truncate tables", err.Error())
	}
}
func (s *SingerTestSuite) TestSuccessCreatePayload() {
	ctx := context.Background()

	singerID := int64(1)
	payload := getCompleteCreatePayload(singerID)

	err := s.provider.Create(ctx, payload)
	assert.Nil(s.T(), err)

	row, err := s.provider.Client.Single().ReadRow(ctx, singerTableName, googleSpanner.Key{singerID}, singerColumns)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), row)

	var singerRow SingerRow
	err = row.ToStruct(&singerRow)
	assert.Nil(s.T(), err)

	assert.Equal(s.T(), payload.SingerID, singerRow.SingerID)
	assert.Equal(s.T(), payload.FirstName, singerRow.FirstName.String())
	assert.Equal(s.T(), payload.LastName, singerRow.LastName.String())
	assert.Equal(s.T(), payload.BirthDate.Year(), singerRow.BirthDate.Date.Year)
	assert.Equal(s.T(), payload.BirthDate.Month(), singerRow.BirthDate.Date.Month)
	assert.Equal(s.T(), payload.BirthDate.Day(), singerRow.BirthDate.Date.Day)

	err = s.provider.Create(ctx, payload)
	assert.NotNil(s.T(), err)
	assert.True(s.T(), errors.Is(err, singer.ErrDuplicate))
}

func (s *SingerTestSuite) TestList() {
	ctx := context.Background()

	singer1 := getCompleteCreatePayload(int64(1))
	singer1.FirstName = "Random"

	singer2 := getCompleteCreatePayload(int64(2))
	singer2.BirthDate = time.Date(2002, 02, 02, 0, 0, 0, 0, time.Local)

	err := s.provider.Create(ctx, singer1)
	if !assert.Nil(s.T(), err) {
		assert.FailNow(s.T(), "failed to create Singer")
	}

	err = s.provider.Create(ctx, singer2)
	assert.Nil(s.T(), err)

	if !assert.Nil(s.T(), err) {
		assert.FailNow(s.T(), "failed to create Singer")
	}

	// Test No Filter
	singers, err := s.provider.List(ctx, singer.FilterPayload{})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 2, len(singers))

	// Test Filter By Name
	singers, err = s.provider.List(ctx, singer.FilterPayload{
		Name: "and",
	})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 1, len(singers))
	assert.Equal(s.T(), int64(1), singers[0].SingerID)

	// Test Filter By BirthDate
	singers, err = s.provider.List(ctx, singer.FilterPayload{
		BirthDateStart: time.Date(2000, 01, 01, 0, 0, 0, 0, time.Local),
	})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 1, len(singers))
	assert.Equal(s.T(), int64(2), singers[0].SingerID)

	// Test Filter By Name and BirthDate
	singers, err = s.provider.List(ctx, singer.FilterPayload{
		Name:           "and",
		BirthDateStart: time.Date(2000, 01, 01, 0, 0, 0, 0, time.Local),
	})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 0, len(singers))

	singers, err = s.provider.List(ctx, singer.FilterPayload{
		Name:         "and",
		BirthDateEnd: time.Date(2000, 01, 01, 0, 0, 0, 0, time.Local),
	})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 1, len(singers))
	assert.Equal(s.T(), int64(1), singers[0].SingerID)
}

func (s *SingerTestSuite) TestGet() {
	ctx := context.Background()

	singerID := int64(1)
	payload := getCompleteCreatePayload(singerID)

	err := s.provider.Create(ctx, payload)
	assert.Nil(s.T(), err)

	result, err := s.provider.Get(ctx, singerID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), result)
	assert.Equal(s.T(), singerID, result.SingerID)
	assert.Equal(s.T(), payload.FirstName, result.FirstName)
	assert.Equal(s.T(), payload.LastName, result.LastName)
	assert.Equal(s.T(), payload.BirthDate.Year(), result.BirthDate.Year())
	assert.Equal(s.T(), payload.BirthDate.Month(), result.BirthDate.Month())
	assert.Equal(s.T(), payload.BirthDate.Day(), result.BirthDate.Day())
	assert.Equal(s.T(), payload.Info.Songs, result.Info.Songs)
	assert.Equal(s.T(), payload.Info.Awards, result.Info.Awards)

	result, err = s.provider.Get(ctx, int64(2000))
	assert.NotNil(s.T(), err)
	assert.True(s.T(), errors.Is(err, singer.ErrNotFound))
	assert.Nil(s.T(), result)
}

func getCompleteCreatePayload(singerID int64) singer.CreatePayload {
	birthDate := time.Date(1990, 7, 21, 0, 0, 0, 0, time.Local)
	completePayload := singer.CreatePayload{
		SingerID:  singerID,
		FirstName: "First",
		LastName:  "Last",
		Info: singer.Info{
			Songs:  []string{"song-1", "song-2"},
			Awards: []string{"awards-1", "awards-2"},
		},
		BirthDate: birthDate,
	}

	return completePayload
}
