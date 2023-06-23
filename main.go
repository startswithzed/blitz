package main

import (
	"github.com/startswithzed/web-ruckus/cmd"
	"log"
)

func main() {
	rootCmd := cmd.GetRootCmd()
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
