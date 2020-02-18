# OpenTelemetry Package for the Gorm ORM #

This package is meant to simplify wrapping Gorm requests to databases with OpenTelemetry Tracing Spans. The functionality within the package is as of OpenTelemetry-go v0.2.1 and is subject to change fairly rapidly as the standard is evolved.

This package is B.Y.O.E. (Bring Your Own Exporter)

Metrics support coming soon!

## Example Usage ##

```golang
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jdefrank/otgorm"

	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"
	_ "github.com/joho/godotenv/autoload"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/exporter/trace/jaeger"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type user struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	FirstName string    `json:"firstName"`
	LastiName string    `json:"lastName"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func readBody(bodyreader io.ReadCloser) (data []byte, err error) {
	body, err := ioutil.ReadAll(io.LimitReader(bodyreader, 1048576))
	if err != nil {
		return nil, err
	}
	if err := bodyreader.Close(); err != nil {
		return nil, err
	}
	return body, nil
}

// NOTE: I've found that if you'd like to separate services in the Jaeger UI,
// you'll need to create multiple exporters which in turn will show you different colors
// per service.
func initTracer() func() {
	//Create Jaeger exporter
	exporter, err := jaeger.NewExporter(
		jaeger.WithCollectorEndpoint(fmt.Sprintf("https://%s/api/traces", os.Getenv("JAEGER_HOST"))),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: "go-otel-gorm",
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	// For demoing purposes, always sample. In a production application, you should
	// configure this to a trace.ProbabilitySampler set at the desired
	// probability.
	tp, err := sdktrace.NewProvider(
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithSyncer(exporter))
	if err != nil {
		log.Fatal(err)
	}
	global.SetTraceProvider(tp)
	return func() {
		exporter.Flush()
	}
}

func main() {
	//Setup the exporter and defer close until main exits
	fn := initTracer()
	defer fn()

	// Connect to database
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)
	db, err := gorm.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	//Register callbacks for GORM, while also passing in config Opts
	otgorm.RegisterCallbacks(db, global.TraceProvider().Tracer("component-gorm"), otgorm.Query(true))

	//Run migration and create a record
	db.AutoMigrate(user{})
	newUser := user{
		FirstName: "John",
		LastiName: "Smith",
	}
	err = db.Create(&newUser).Error
	if err != nil {
		log.Print(err)
	}

	//Create router
	r := chi.NewRouter()

	//Register Endpoints for the API
	r.Post("/user", func(w http.ResponseWriter, r *http.Request) {
		orm := otgorm.WithContext(r.Context(), db)
		var u user
		body, err := readBody(r.Body)
		if err != nil {
			log.Print(err)
			return
		}
		err = json.Unmarshal(body, &u)
		if err != nil {
			log.Print(err)
			return
		}
		err = orm.Create(&u).Error
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode("{\"Error\":\"" + err.Error() + "\"")
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	})
	http.ListenAndServe(":3000", r)
}
```

## License ##
The MIT License (MIT). Please see [License File](LICENSE) for more information.
