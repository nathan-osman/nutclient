## nutclient

[![Go Reference](https://pkg.go.dev/badge/github.com/nathan-osman/nutclient.svg)](https://pkg.go.dev/github.com/nathan-osman/nutclient)
[![MIT License](https://img.shields.io/badge/license-MIT-9370d8.svg?style=flat)](https://opensource.org/licenses/MIT)

This package provides a very simple [NUT](https://networkupstools.org/) client for Go.

Go 1.18 is the minimum supported version.

### Usage

To use the package in your program, begin by importing it:

```golang
import "github.com/nathan-osman/nutclient/v2"
```

Next, create a `Client` instance using `nutclient.New()`.

The `Config` struct passed to `New` specifies the address of the NUT server and the name of the UPS you are connecting to. It also allows you to specify the poll interval and callbacks that will be invoked when power events occur:

```golang
c := nutclient.New(&nutclient.Config{
    Addr: "localhost:3493",
    Name: "ups",
    PollInterval: 5 * time.Second,
    ConnectedFn: func() {
        fmt.Println("Connected!")
    },
    DisconnectedFn: func() {
        fmt.Println("Disconnected!")
    },
    PowerLostFn: func() {
        fmt.Println("Power lost!")
    },
    PowerRestoredFn: func() {
        fmt.Println("Power restored!")
    },
})
defer c.Close()
```

The above will connect to the NUT server and monitor the status every 5 seconds and invoke the associated callbacks when the status changes.

To manually obtain status values, use `Status()`. For example, to lookup the current battery status, use:

```golang
l, err := c.Status()
if err != nil {
    // TODO: handle error
}
fmt.Printf(
    "Battery: %s\n",
    l["ups.status"],
)
```

> Note: if the client is not currently connected to the NUT server, the method will return an error.
