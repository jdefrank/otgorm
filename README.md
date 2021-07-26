# OpenTelemetry Package for the Gorm ORM V1 #

This package is meant to simplify wrapping Gorm requests to databases with OpenTelemetry Tracing Spans. The functionality within the package is as of OpenTelemetry-go v1.0.0-RC1 and is subject to change fairly rapidly as the standard is evolved.

This package is B.Y.O.E. (Bring Your Own Exporter)

Metrics support coming soon!

## Example Usage ##

Make sure you have the following:

- Docker
- Go 1.16
- cURL or Postman for testing

Run the following commands to create the testing environment:
- `docker run -d -p 5432:5432 -e POSTGRES_USER=testuser -e POSTGRES_PASSWORD=password! -e POSTGRES_DB=test --name postgres postgres:alpine`
- `docker run -d --name jaeger -e COLLECTOR_ZIPKIN_HTTP_PORT=9411 -p 5775:5775/udp -p 6831:6831/udp -p 6832:6832/udp -p 5778:5778 -p 16686:16686 -p 14268:14268 -p 14250:14250 -p 9411:9411 jaegertracing/all-in-one:1.16`

### .env File ###

```
JAEGER_HOST=127.0.0.1
DB_USER=testuser
DB_PASS=password!
DB_HOST=127.0.0.1
DB_PORT=5432
DB_NAME=test
DB_SSLMODE=disable
```

### Example App ###

```golang
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/yuraxdrumz/otgorm"

	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/joho/godotenv/autoload"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/exporter/trace/jaeger"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type user struct {
	ID        uint      `gorm:"primary_key" json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
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

func httpTraceWrapper(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		t := global.TraceProvider().Tracer("component-http")
		ctx, span := t.Start(r.Context(), r.URL.Path)
		r = r.WithContext(ctx)
		h.ServeHTTP(w, r)
		span.End()
	}
	return http.HandlerFunc(fn)
}

// NOTE: I've found that if you'd like to separate services in the Jaeger UI,
// you'll need to create multiple exporters which in turn will show you different colors
// per service.
func initTracer() func() {
	//Create Jaeger exporter
	exporter, err := jaeger.NewExporter(
		jaeger.WithCollectorEndpoint(fmt.Sprintf("http://%s:14268/api/traces", os.Getenv("JAEGER_HOST"))),
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
	connString := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_SSLMODE"),
	)
	db, err := gorm.Open("postgres", connString)
	if err != nil {
		panic(err)
	}

	//Register callbacks for GORM, while also passing in config Opts
	otgorm.RegisterCallbacks(db,
		otgorm.WithTracer(global.TraceProvider().Tracer("component-gorm")),
		otgorm.Query(true),
		otgorm.AllowRoot(true),
	)

	//Run migration and create a record
	db.AutoMigrate(user{})
	newUser := user{
		FirstName: "John",
		LastName:  "Smith",
	}
	//Since this first DB call is outside of a parent,
	//lets set up empty context and the DB client with that context
	ctx := context.Background()
	orm := otgorm.WithContext(ctx, db)
	err = orm.Create(&newUser).Error
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
	http.ListenAndServe(":3000", httpTraceWrapper(r))
}
```

## License ##
The MIT License (MIT). Please see [License File](LICENSE) for more information.
