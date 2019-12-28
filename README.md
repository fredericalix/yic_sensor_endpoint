# Sensor

Rest API endpoint to send your sensors updates.

You must have a sensor token, to get one, see auth service API.

## Request

    curl -X POST                                                                 \
       -H "Authorization: Bearer TfBg8RxcHzTOiAazX07pEE17VqwI_4IneURpWwNxtGs"    \
       -H "Content-Type: application/json"                                       \
       http://localhost:2020/sensors                                             \
       -d '{"id":"1377959e-97ce-46c1-9715-22c34bb9afbe", "propeler":"slow", "on_fire": false, "grey": false}'

### Success response

    Code: 200 OK

### Error response

    Code: 401 Unauthorized
    Content: application/json
    {"message": "invalid token"}

    Code 400 Bad Request
    Content: application/json
    {"message":"missing id"}

## Compile & run

    go build
    ./sensor-endpoint

## Exemple config in Environment variables

``` sh
    RABBITMQ_URI="amqp://guest:guest@rabbit:5672"
    AUTH_CHECK_URI="http://auth:1234/auth/check"
    PORT=8080
```

## Generate of the swagger doc

Install go-swagger <https://goswagger.io/install.html> then generate the swagger specification

    swagger generate spec -o swagger_sensor.json

To quick show the doc

    swagger serve swagger_sensor.json
