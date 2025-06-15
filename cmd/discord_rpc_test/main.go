package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IanWorley/presence-go/pkg"
)

func main() {
	client := pkg.NewClient("1383078278478303474")

	err := client.Connect()
	if err != nil {
		fmt.Println("Failed to connect to Discord RPC:", err)
		return
	}

	fmt.Println("Discord RPC handshake succeeded!")

	client.Send(pkg.Status{
		V:        1,
		ClientID: "1383078278478303474",
		Cmd:      "SET_ACTIVITY",
		Args: pkg.Args{
			PID: os.Getpid(),
			Activity: &pkg.Activity{
				Details: "build a go discord rpc library",
				State:   "testing",
				Timestamps: &pkg.Timestamps{
					Start: int(time.Now().UTC().Unix()),
				},
			},
		},
		Nonce: "1234567890",
	})

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	client.Disconnect()
	fmt.Println("Discord RPC disconnected!")
}
