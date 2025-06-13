package main

import (
	"fmt"
	"time"

	"github.com/IanWorley/presence-go/pkg"
)

func main() {
	client := pkg.NewClient("1383078278478303474")

	go func() {
		err := client.Connect()
		if err != nil {
			fmt.Println("Failed to connect to Discord RPC:", err)
			return
		}
	}()

	time.Sleep(10 * time.Second)

	client.Send(map[string]any{
		"state": "Testing",
	})

	fmt.Println("Discord RPC handshake succeeded!")

	select {} // Keep the program running
}
