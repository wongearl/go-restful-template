package main

import (
	"log"

	"github.com/wongearl/go-restful-template/cmd/ai-server/app"

	"github.com/spf13/cobra"
)

func main() {
	var cmd *cobra.Command
	cmd = app.NewAIServerCommand()

	if err := cmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}
