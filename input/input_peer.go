package main
import (
        //"github.com/apanda/smpc/core"
        sproto "github.com/apanda/smpc/proto"
        //"github.com/apanda/go-zmq"
        "fmt"
        "flag"
        "os"
        "io/ioutil"
        "encoding/json"
        "code.google.com/p/goprotobuf/proto"
        )
type Configuration struct {
    Clients []string
    Shell   bool
}
func main() {
    config := flag.String("config", "conf", "Configuration file")
    flag.Parse()
    fmt.Printf ("Starting with configuration %s\n", *config)
    contents, err := ioutil.ReadFile(*config)
    if err != nil {
        fmt.Printf ("Could not read configuration file, error = %s", err)
        os.Exit(1)
    }
    var configStruct Configuration
    err = json.Unmarshal(contents, &configStruct)
    if err != nil {
        fmt.Println("Error reading json file: ", err)
        os.Exit(1)
    }
    fmt.Printf("Shell %t\n", configStruct.Shell)
    for k, v := range configStruct.Clients {
        fmt.Printf("%d %s\n", k, v)
    }
    action := new(sproto.Action)
    t := sproto.Action_Set
    action.Action = &t
    res := "food"
    action.Result = &res
    buffer, err := proto.Marshal(action)
    var _ = buffer
    if err != nil {
        fmt.Println("error: ", err)
    }
}

