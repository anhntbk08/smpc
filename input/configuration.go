/* Configuration tools for input peer */
package main
import (
        "io/ioutil"
        "encoding/json"
        "fmt"
        )
type Configuration struct {
    PubAddress string
    ControlAddress string
    Clients int
    Shell   bool
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

