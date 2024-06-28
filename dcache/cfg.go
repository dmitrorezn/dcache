package main

import (
	"flag"

	"github.com/joho/godotenv"
)

var nodeID = flag.String("nodeID", "leader", "Node ID env file suffix")

func init() {
	flag.Parse()

	if err := godotenv.Load(".env_" + *nodeID); err != nil {
		panic(err)
	}
}
