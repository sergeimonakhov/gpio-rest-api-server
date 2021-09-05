## GPIO rest api server

Activate and deactivate gpio pins

## Rest-api schema
```bash
POST '{"active":true}' /gpios/{GPIO_PIN_ID}
GET /gpios/{GPIO_PIN_ID}, response: {"is_active":true}
```

## Integrate with [home-assistant](https://www.home-assistant.io)
```yaml
switch:
  - platform: rest
    name: GPIO2
    resource: http://{GPIO_REST_API_SERVER_ADDR}/gpios/2
    body_on: '{"active": true}'
    body_off: '{"active": false}'
    is_on_template: "{{ value_json.is_active }}"
    headers:
      Content-Type: application/json
    verify_ssl: false
```

## Requirements:

* [support gpiomem](https://github.com/stianeikeland/go-rpio#using-without-root)

## Tested and working on:

* Raspberry Pi 3 Model B Rev 1.2 (Raspbian GNU/Linux 10 (buster))

## Flags:

```bash
Usage of ./gpio-api:
  -dbfile string
    	Set db file (default "./gpio.db")
  -listen-port int
    	Set http server listen port (default 8081)
  -recovery
    	Recovery gpio state at start
```
