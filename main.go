package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
	"github.com/jessevdk/go-flags"
)

type CliArgs struct {
	ConfigFile string `short:"c" long:"config" env:"QBTAGARR_CONFIG_FILE" description:"Path to config file" required:"true"`
}

type Config struct {
	Server string `yaml:"server" validate:"required"`
}

func main() {
	log.Println("starting qbtagarr")

	args := CliArgs{}

	_, err := flags.Parse(&args)
	if err != nil {
		os.Exit(1)
	}

	_, err = os.Stat(args.ConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	content, err := os.ReadFile(args.ConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	cfg := Config{}

	err = yaml.Unmarshal(content, &cfg)
	if err != nil {
		log.Fatal(err)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err = validate.Struct(cfg); err != nil {
		log.Fatal(err)
	}

	r := gin.Default()
	r.POST("/api/webhook", func(c *gin.Context) {
		infoHash := c.PostForm("infoHash")

		if infoHash == "" {
			log.Println("Error: missing infoHash")
			c.AbortWithStatus(http.StatusBadRequest)
		}

		log.Println(infoHash)
	})

	if err = r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
