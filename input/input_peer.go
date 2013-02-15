package main
import (
        //"github.com/apanda/smpc/core"
        sproto "github.com/apanda/smpc/proto"
        zmq "github.com/apanda/go-zmq"
        "fmt"
        "flag"
        "os"
        "os/signal"
        "io/ioutil"
        "encoding/json"
        "code.google.com/p/goprotobuf/proto"
        )
type Configuration struct {
    PubAddress string
    ControlAddress string
    Clients []string
    Shell   bool
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
    fmt.Printf("Shell %t\n", configStruct.Shell)
    for k, v := range configStruct.Clients {
        fmt.Printf("%d %s\n", k, v)
    }
    // Create the 0MQ context
    ctx, err := zmq.NewContext()
    if err != nil {
        fmt.Println("Error creating 0mq context: ", err)
        os.Exit(1)
    }
    // Establish the PUB-SUB connection that will be used to direct all the computation clusters
    pubsock, err := ctx.Socket(zmq.Pub)
    if err != nil {
        fmt.Println("Error creating PUB socket: ", err)
        os.Exit(1)
    }
    err = pubsock.Bind(configStruct.PubAddress)
    if err != nil {
        fmt.Println("Error binding PUB socket: ", err)
        os.Exit(1)
    }
    // Establish coordination socket
    coordsock, err := ctx.Socket(zmq.Rep)
    if err != nil {
        fmt.Println("Error creating REP socket: ", err)
        os.Exit(1)
    }
    err = coordsock.Bind(configStruct.ControlAddress)
    if err != nil {
        fmt.Println("Error creating  ", err)
        os.Exit(1)
    }
    connectedSoFar := 0
    fmt.Printf("Waiting for %d connections\n", len(configStruct.Clients))
    
    for connectedSoFar < len(configStruct.Clients) {
        syncMsg, err := coordsock.Recv()
        if err != nil {
            fmt.Println("Error receiving on coordination socket", err)
            os.Exit(1)
        }
        var _ = syncMsg
        connectedSoFar += 1
        fmt.Printf("Waiting for %d connections\n", len(configStruct.Clients) - connectedSoFar)
        resp := make([][]byte, 1)
        resp[0] = make([]byte, 1)
        err = coordsock.Send(resp)
        if err != nil {
            fmt.Println("Error receiving on coordination socket", err)
            os.Exit(1)
        }
    }
    q <- 0
    defer func() {
        pubsock.Close()
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
    // TODO: THIS IS JUST TEST CODE THAT SHOULD BE DELETED
    action := new(sproto.Action)
    t := sproto.Action_Set
    action.Action = &t
    res := "food"
    action.Result = &res
    buffer, err := proto.Marshal(action)
    if err != nil {
        fmt.Println("error: ", err)
        os.Exit(1)
    }
    unaction := &sproto.Action{}
    err = proto.Unmarshal(buffer, unaction)
    if err != nil {
        fmt.Println("error: ", err)
        os.Exit(1)
    }
    if *unaction.Result != *action.Result {
        fmt.Println("Comparison failed")
    }
    // TODO: END THIS IS JUST TEST CODE THAT SHOULD BE DELETED
    go EventLoop(config, end_channel)
    select {
        case <- os_channel:
        case <- end_channel: 
    }
    // <-signal_channel
    os.Exit(0)
}

