## nutclient

[![Go Reference](https://pkg.go.dev/badge/github.com/nathan-osman/nutclient.svg)](https://pkg.go.dev/github.com/nathan-osman/nutclient)
[![MIT License](https://img.shields.io/badge/license-MIT-9370d8.svg?style=flat)](https://opensource.org/licenses/MIT)

This package provides a very simple [NUT](https://networkupstools.org/) client for Go.

### Usage

To use the package in your program, begin by importing it:

```golang
import "github.com/nathan-osman/nutclient"
```

Next, create a `Client` instance using `nutclient.New()`.

The `Config` struct passed to `New` specifies the address of the NUT server and the name of the UPS you are connecting to. It also allows you to specify the callbacks that will be invoked when power events occur:

```golang
c := nutclient.New(&nutclient.Config{
    Addr: "localhost:3493",
    Name: "ups",
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
