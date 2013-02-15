package main
import (
        //"github.com/apanda/smpc/core"
        zmq "github.com/apanda/go-zmq"
        "fmt"
        "flag"
        "os"
        "os/signal"
        "io/ioutil"
        "encoding/json"
        )
type Configuration struct {
    PubAddress string
    ControlAddress string
}
func EventLoop (config *string, q chan int) {
    fmt.Printf ("Starting with configuration %s\n", *config)
    // Read the configuration file
    contents, err := ioutil.ReadFile(*config)
    if err != nil {
        fmt.Printf ("Could not read configuration file, error = %s", err)
        os.Exit(1)
    }
    var configStruct Configuration
    // Parse configuration, produce an object. We assume configuration is in JSON
    err = json.Unmarshal(contents, &configStruct)
    if err != nil {
        fmt.Println("Error reading json file: ", err)
        os.Exit(1)
    }
    // Create the 0MQ context
    ctx, err := zmq.NewContext()
    if err != nil {
        fmt.Println("Error creating 0mq context: ", err)
        os.Exit(1)
    }
    // Establish the PUB-SUB connection that will be used to direct all the computation clusters
    subsock, err := ctx.Socket(zmq.Sub)
    if err != nil {
        fmt.Println("Error creating PUB socket: ", err)
        os.Exit(1)
    }
    err = subsock.Connect(configStruct.PubAddress)
    if err != nil {
        fmt.Println("Error binding PUB socket: ", err)
        os.Exit(1)
    }
    // Establish coordination socket
    coordsock, err := ctx.Socket(zmq.Req)
    if err != nil {
        fmt.Println("Error creating REP socket: ", err)
        os.Exit(1)
    }
    err = coordsock.Connect(configStruct.ControlAddress)
    if err != nil {
        fmt.Println("Error creating  ", err)
        os.Exit(1)
    }
    resp := make([][]byte, 1)
    resp[0] = make([]byte, 1)
    err = coordsock.Send(resp)
    if err != nil {
        fmt.Println("Error sending on coordination socket", err)
        os.Exit(1)
    }
    syncmsg, err := coordsock.Recv()
    if err != nil {
        fmt.Println("Error receiving on coordination socket", err)
    }
    var _ = syncmsg
    q <- 0
    defer func() {
        subsock.Close()
        coordsock.Close()
        ctx.Close()
        fmt.Println("Closed socket")
    }()
}

func main() {
    // Start up by setting up a flag for the configuration file
    config := flag.String("config", "conf", "Configuration file")
    flag.Parse()
    os_channel := make(chan os.Signal)
    signal.Notify(os_channel)
    end_channel := make(chan int)
    go EventLoop(config, end_channel)
    select {
        case <- os_channel:
        case <- end_channel: 
    }
    // <-signal_channel
    os.Exit(0)
}

