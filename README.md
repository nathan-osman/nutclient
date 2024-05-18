## nutclient

[![Go Reference](https://pkg.go.dev/badge/github.com/nathan-osman/nutclient.svg)](https://pkg.go.dev/github.com/nathan-osman/nutclient)
[![MIT License](https://img.shields.io/badge/license-MIT-9370d8.svg?style=flat)](https://opensource.org/licenses/MIT)

This package provides a very simple [NUT](https://networkupstools.org/) client for Go.

Go 1.18 is the minimum supported version.

### Basic Usage

To use the package in your program, begin by importing it:

```golang
import "github.com/nathan-osman/nutclient/v3"
```

Next, create a `Client` instance using `nutclient.New()`:

```golang
c := nutclient.New(nil)
defer c.Close()
```

The `Config` struct passed to `New` can be used to specify the address of the NUT server. It also allows you to specify the callbacks that will be invoked when power events occur:

```golang
c := nutclient.New(
    &nutclient.Config{
        Addr: "localhost:3493",
        ConnectedFn: func() {
            fmt.Println("Connected!")
        },
        DisconnectedFn: func() {
            fmt.Println("Disconnected!")
        },
    },
)
defer c.Close()
```

Once connected, you can use methods like `Get()` to interact with the NUT server. For example, to lookup the current battery status of the UPS named "ups", you would use:

```golang
v, err := c.Get("VAR ups ups.status")
if err != nil {
    // TODO: handle error
}
fmt.Printf("Battery: %s\n", v)
```

> Note: if the client is not currently connected to the NUT server, the method will return an error.

### Monitoring a UPS

The `monitor` package simplifies the task of monitoring a UPS server for power events. Its usage is fairly straightforward:

```golang
import "github.com/nathan-osman/nutclient/v3/monitor"

c := monitor.New(
    &monitor.Config{
        Addr: "localhost:3493",
        Name: "ups",
        PowerLostFn: func() {
            fmt.Println("Power lost!")
        },
        PowerRestoredFn: func() {
            fmt.Println("Power restored!")
        },
    },
)
defer c.Close()
```
