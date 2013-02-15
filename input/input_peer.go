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
        "time"
        )
type Configuration struct {
    PubAddress string
    ControlAddress string
    Clients int
    Shell   bool
}
func SendHello (sock *zmq.Socket, exitChannel chan int, notificationChannel chan bool) {
    sleepChan := make(chan bool, 1)
    for true {
        msg := make([][]byte, 1)
        msg[0] = []byte("HELO")
        err := sock.Send(msg)
        if err != nil {
            fmt.Println("Error sending HELO message on socket", err)
            exitChannel <- 1
            return
        }
        go func() {
            time.Sleep(10 * time.Millisecond)
            sleepChan <- true
        }()
        select {
            case <- notificationChannel:
                return
            case <- sleepChan:
            // Do nothing
        }
    }
}

/* Read and parse the configuration file */
func ParseConfig (config *string, q chan int) (*Configuration) {
    fmt.Printf ("Starting with configuration %s\n", *config)
    // Read the configuration file
    contents, err := ioutil.ReadFile(*config)
    if err != nil {
        fmt.Printf ("Could not read configuration file, error = %s", err)
        q <- 1
        return nil
    }
    var configStruct Configuration
    // Parse configuration, produce an object. We assume configuration is in JSON
    err = json.Unmarshal(contents, &configStruct)
    if err != nil {
        fmt.Println("Error reading json file: ", err)
        q <- 1
        return nil
    }
    return &configStruct
}

/* The main event loop */
func EventLoop (config *string, q chan int) {
    configStruct := ParseConfig(config, q)
    // Create the 0MQ context
    ctx, err := zmq.NewContext()
    if err != nil {
        fmt.Println("Error creating 0mq context: ", err)
        q <- 1
        return
    }
    // Establish the PUB-SUB connection that will be used to direct all the computation clusters
    pubsock, err := ctx.Socket(zmq.Pub)
    if err != nil {
        fmt.Println("Error creating PUB socket: ", err)
        q <- 1
        return
    }
    err = pubsock.Bind(configStruct.PubAddress)
    if err != nil {
        fmt.Println("Error binding PUB socket: ", err)
        q <- 1
        return
    }
    // Establish coordination socket
    coordsock, err := ctx.Socket(zmq.Rep)
    if err != nil {
        fmt.Println("Error creating REP socket: ", err)
        q <- 1
        return
    }
    err = coordsock.Bind(configStruct.ControlAddress)
    if err != nil {
        fmt.Println("Error creating  ", err)
        q <- 1
        return
    }
    connectedSoFar := 0
    fmt.Printf("Waiting for %d connections\n", configStruct.Clients)
    // Make sure the notification channel cannot buffer messages, this is some what like a process.join
    notification := make(chan bool, 0)
    // Start a go routine to send HELO
    go SendHello(pubsock, q, notification) 
    
    for connectedSoFar < configStruct.Clients {
        _, err := coordsock.Recv()
        if err != nil {
            fmt.Println("Error receiving on coordination socket", err)
            q <- 1
            return
        }
        connectedSoFar += 1
        fmt.Printf("Waiting for %d connections\n", configStruct.Clients - connectedSoFar)
        resp := make([][]byte, 1)
        resp[0] = make([]byte, 1)
        err = coordsock.Send(resp)
        if err != nil {
            fmt.Println("Error receiving on coordination socket", err)
            q <- 1
            return
        }
    }
    // Tell the HELO routine to stop
    notification <- true
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
    end_channel := make(chan int, 1)
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
    var status = 0
    select {
        case <- os_channel:
        case status = <- end_channel: 
    }
    os.Exit(status)
}

