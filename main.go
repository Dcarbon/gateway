package main

import (
	"log"

	"github.com/Dcarbon/gateway/serve"
	"github.com/Dcarbon/go-shared/libs/utils"
)

var docFile = utils.StringEnv("DOC_FILE", "static/api.swagger.json")

// var docFile = utils.StringEnv("DOC_FILE", "static/petstore.json")

func main() {
	var port = 4090
	mux, err := serve.NewServeMux(docFile)
	utils.PanicError("Create mux", err)

	log.Fatalln(mux.Start(port))
}
