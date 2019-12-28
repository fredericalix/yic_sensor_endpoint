// Sensors service
//
//     Schemes: http, https
//     Host: localhost:2020
//     Version: 0.0.1
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//
// swagger:meta
package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"

	auth "github.com/fredericalix/yic_auth"

	_ "net/http/pprof"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

type handler struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func main() {
	viper.AutomaticEnv()
	viper.SetDefault("PORT", "8080")

	configFile := flag.String("config", "./config.toml", "path of the config file")
	flag.Parse()
	viper.SetConfigFile(*configFile)
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Printf("cannot read config file: %v\nUse env instead\n", err)
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.GET("/sensors/_health", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.GET("/sensors/debug/pprof/*", echo.WrapHandler(http.DefaultServeMux), middleware.Rewrite(map[string]string{"/sensors/*": "/$1"}))

	h := handler{}
	h.conn, err = amqp.Dial(viper.GetString("RABBITMQ_URI"))
	failOnError(err, "Failed to connect to RabbitMQ")

	roles := auth.Roles{"sensor": "w"}
	sessions := auth.NewValidHTTP(viper.GetString("AUTH_CHECK_URI"))
	sessions, err = auth.NewValidAMQPCache(h.conn, roles, sessions)
	failOnError(err, "Failed to create auth AMQP cache")
	e.POST("/sensors", h.sensor, auth.Middleware(sessions, roles))

	go func() {
		log.Fatalf("closing: %s", <-h.conn.NotifyClose(make(chan *amqp.Error)))
	}()

	h.ch, err = h.conn.Channel()
	failOnError(err, "Failed to open a channel")

	err = h.ch.ExchangeDeclare(
		"sensors", // name
		"topic",   // type
		true,      // durable
		false,     // auto-deleted
		false,     // internal
		false,     // no-wait
		nil,       // arguments
	)
	failOnError(err, "Failed to declare an exchange")

	q, err := h.ch.QueueDeclare(
		"sensors", // name
		true,      // durable
		false,     // delete when usused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	failOnError(err, "Failed to declare a queue")

	// Get every routing key
	err = h.ch.QueueBind(
		q.Name,    // queue name
		"#",       // routing key
		"sensors", // exchange
		false,     // no wait
		nil,       // args
	)
	failOnError(err, "Failed to bind a queue")

	// start the server
	host := ":" + viper.GetString("PORT")
	tlscert := viper.GetString("TLS_CERT")
	tlskey := viper.GetString("TLS_KEY")
	if tlscert == "" || tlskey == "" {
		e.Logger.Error("No cert or key provided. Start server using HTTP instead of HTTPS !")
		e.Logger.Fatal(e.Start(host))
	}
	e.Logger.Fatal(e.StartTLS(host, tlscert, tlskey))
}

// swagger:parameters authAPIKey sensors
type authAPIKey struct {
	// Your yourITcity API Key (Bearer <your_sensor_token>)
	//in: header
	//required: true
	Authorization string
}

//swagger:parameters sensorRequest sensors
type sensorRequest struct {
	//in:body
	//example: {"id":"cd0a6b8a-a32f-4cec-bd4d-38b24ac793e0", "activity":"normal"}
	Body struct {
		// Sensor UUID
		// required: true
		// example: cd0a6b8a-a32f-4cec-bd4d-38b24ac793e0
		ID string `json:"id"`

		// example: value1
		Key1 string `json:"key1"`
		// example: value2
		Key2 string `json:"key2"`
		// example: value3
		Key3 string `json:"key3"`
	}
}

// swagger:route POST /sensors sensors
//
// Sensors
//
// Send your sensors update. The json with the sensors info must at least contains an id of type uuid.
//
// Consumes:
//   application/json
// Produces:
//   application/json
// Schema:
//   http, https
// Responces:
//   200: Success
//   401: Unauthorized
//   400: Bad Request
func (h *handler) sensor(c echo.Context) error {
	coorID := randID()

	// Auth
	a := c.Get("account").(auth.Account)
	aid := a.ID.String()

	// Validate json sensor
	var payload map[string]interface{}
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": err.Error()})
	}
	if payload == nil || len(payload) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "empty payload"})
	}
	sid, ok := payload["id"].(string)
	if !ok && sid == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "missing id field"})
	}
	_, err := uuid.FromString(sid)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "id must by UUID v4"})
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": err.Error()})
	}

	err = h.ch.Publish(
		"sensors",   // exchange
		aid+"."+sid, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			DeliveryMode:  amqp.Persistent,
			CorrelationId: coorID,
			Headers:       amqp.Table{"timestamp": time.Now().Format(time.RFC3339Nano)},
			ContentType:   "application/json",
			Body:          body,
		})
	if err != nil {
		c.Logger().Errorf("sensor: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

func randID() string {
	return base64.RawURLEncoding.EncodeToString(uuid.Must(uuid.NewV4()).Bytes())
}
