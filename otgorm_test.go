package otgorm_test

import (
	"context"
	"database/sql"
	"github.com/jdefrank/otgorm"
	"github.com/jinzhu/gorm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel/oteltest"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"testing"
)

func TestTrace(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Trace Suite")
}

type User struct {
	name string
}

var _ = Describe("Trace", func() {
	var db *gorm.DB
	var mock sqlmock.Sqlmock
	var sr *oteltest.StandardSpanRecorder
	var tp trace.TracerProvider
	BeforeEach(func() {
		var err error
		var tmpDb *sql.DB
		tmpDb, mock, err = sqlmock.New()
		Expect(err).To(BeNil())

		db, err = gorm.Open("postgres", tmpDb)
		Expect(err).To(BeNil())

		db.LogMode(true)

		sr = new(oteltest.StandardSpanRecorder)
		tp = oteltest.NewTracerProvider(oteltest.WithSpanRecorder(sr))
		//Register callbacks for GORM, while also passing in config Opts
		otgorm.RegisterCallbacks(db,
			otgorm.WithTracer(tp.Tracer("gorm")),
			otgorm.Query(true),
			otgorm.AllowRoot(true),
		)
	})
	AfterEach(func() {
		db.Close()
		db = nil
	})
	Context("With no parent span", func() {
		ctx := context.Background()

		It("Should take spans for Create queries", func() {
			db := otgorm.WithContext(ctx, db)
			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO "users"`).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()
			db.Create(&User{name: "foo"})

			completed := sr.Completed()
			Expect(completed).To(HaveLen(1))
			Expect(completed[0].Name()).To(Equal("gorm:create"))
			Expect(completed[0].Attributes()["gorm.query"].AsString()).To(Equal(`INSERT INTO "users" DEFAULT VALUES RETURNING "users".*`))
		})

		It("Should record spans for Delete and then Create queries", func() {
			db := otgorm.WithContext(ctx, db)
			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO "users"`).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()
			mock.ExpectBegin()
			mock.ExpectExec(`DELETE FROM "users"`).
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()
			db.Create(&User{name: "bim"})
			db.Delete(&User{name: "bim"})

			completed := sr.Completed()
			Expect(completed).To(HaveLen(2))
			Expect(completed[0].Name()).To(Equal("gorm:create"))
			Expect(completed[1].Name()).To(Equal("gorm:delete"))
		})
	})

	It("Should work with a parent trace", func() {
		ctx, span := tp.Tracer("test").Start(context.Background(), "myTrace")
		db := otgorm.WithContext(ctx, db)
		mock.ExpectBegin()
		mock.ExpectExec(`INSERT INTO "users"`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()
		db.Create(&User{name: "foo"})

		span.End()

		completed := sr.Completed()
		Expect(completed).To(HaveLen(2))
		Expect(completed[0].Name()).To(Equal("gorm:create"))
		Expect(completed[0].Attributes()["gorm.query"].AsString()).To(Equal(`INSERT INTO "users" DEFAULT VALUES RETURNING "users".*`))
		Expect(completed[1].Name()).To(Equal("myTrace"))
	})
})
