# bifrost-server

This project provides a minimal common framework for servers using the Bifrost (BAPS3) protocol.

**NOTE: This is not yet suitable for production use!**

## Usage

Launching a TCP Bifrost server that responds to `read` and `quit`:

```go
import (                                                                                                                      "github.com/UniversityRadioYork/baps3-go"
        "github.com/UniversityRadioYork/bifrost-server/request"
        "github.com/UniversityRadioYork/bifrost-server/tcpserver"
)                                                                                                                     

func main() {
        // ...
                    
        tcpserver.Serve(request.Map{                                                                                                                                                                                               
                baps3.RqRead: func(b, r chan<- *baps3.Message, s []string) (bool, error) { return handleRead(b, r, t, s) },                                                                                                        
                baps3.RqQuit: func(_, _ chan<- *baps3.Message, _ []string) (bool, error) { return true, nil },                                                                                                                     
        }, "localhost:1234")                                                                                                             
}
```

Better usage instructions to come later.

## Licence

MIT licence; see `LICENSE`.

## Contribution

All pull requests welcome.  Please use `go fmt`.
