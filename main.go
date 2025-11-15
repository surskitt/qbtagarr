package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/autobrr/go-qbittorrent"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
	"github.com/jessevdk/go-flags"
)

type CliArgs struct {
	ConfigFile string `short:"c" long:"config" env:"QBTAGARR_CONFIG_FILE" description:"Path to config file" required:"true"`
}

type Config struct {
	Server   string `yaml:"server" validate:"required"`
	Trackers map[string][]string
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func genWebhookHandler(cfg *Config, qb *qbittorrent.Client) func(c *gin.Context) {
	return func(c *gin.Context) {
		ctx := context.Background()

		if err := qb.LoginCtx(ctx); err != nil {
			log.Fatalf("could not log into client: %q", err)
		}

		infoHash := c.PostForm("infoHash")

		if infoHash == "" {
			log.Println("Error: missing infoHash")
			c.AbortWithStatus(http.StatusBadRequest)
		}

		hashes := []string{infoHash}

		torrents, err := qb.GetTorrents(qbittorrent.TorrentFilterOptions{
			Hashes: hashes,
		})

		if err != nil {
			log.Println("could not get torrents from client: %q", err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}

		if len(torrents) == 0 {
			log.Println("no torrents found")
			c.AbortWithStatus(http.StatusNotFound)
		}

		t := torrents[0]

		u, err := url.Parse(t.Tracker)
		if err != nil {
			log.Println("could not parse tracker url: %q", err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}

		tracker := u.Hostname()

		for trackerName, urls := range cfg.Trackers {
			for _, url := range urls {
				if url == tracker {
					tag := "site:" + trackerName
					qb.AddTags(hashes, tag)
				}
			}
		}
	}
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

	qb := qbittorrent.NewClient(qbittorrent.Config{
		Host: cfg.Server,
	})

	r := gin.Default()
	r.POST("/api/webhook", genWebhookHandler(&cfg, qb))

	if err = r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
