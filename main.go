package main

import (
	"flag"
	"fmt"
	"github.com/christopher-dG/go-obs-websocket"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

type config struct {
	sceneName string
	itemName  string
	endpoints string
	password  string
	port      int
}

func parseFlags() (config, error) {
	c := config{}
	flag.StringVar(&c.endpoints, "endpoints", "", "The OBS endpoints to poll")
	flag.StringVar(&c.password, "password", "", "The OBS password")
	flag.StringVar(&c.sceneName, "scene-name", "Panel", "Name of the scene")
	flag.StringVar(&c.itemName, "item-name", "RTMP stream", "Name of the item")
	flag.IntVar(&c.port, "port", 4444, "OBS port number")
	flag.Parse()

	if c.endpoints == "" {
		return c, fmt.Errorf("you must specify --endpoints")
	}
	return c, nil
}

func poll(endpoint string, port int, password, scene, item string) {
	// Connect a client.
	for {
		func() {
			log.Printf("Connecting to %s:%d...\n", endpoint, port)
			c := obsws.Client{Host: endpoint, Port: port, Password: password}
			if err := c.Connect(); err != nil {
				log.Printf("Couldn't connect to %s:%d, retrying later...\n", endpoint, port)
				time.Sleep(20 * time.Second)
				return
			}
			log.Printf("Connected to %s:%d\n", endpoint, port)
			defer c.Disconnect()

			for {
				req := obsws.NewGetSceneItemPropertiesRequest(scene, item)
				resp, err := req.SendReceive(c)
				if err != nil {
					log.Printf("Couldn't get properties of %s/%s: %v\n", scene, item, err)
					return
				}
				if (resp.Width < 1919 || resp.Width > 1921) && (resp.Height < 1079 || resp.Height > 1081) {
					scaleW := 1920.0 / float64(resp.SourceWidth)
					scaleH := 1080.0 / float64(resp.SourceHeight)
					scale := scaleH
					if scaleW < scaleH {
						scale = scaleW
					}
					// This sleep prevents us from actually crashing OBS completely. Yes, really.
					time.Sleep(3 * time.Second)
					log.Printf("Scaling %s/%s by %f (current size: %fx%f; source size: %dx%d)\n", scene, item, scale, resp.Width, resp.Height, resp.SourceWidth, resp.SourceHeight)
					//obsws.NewSetSceneItemPropertiesRequest(scene, item, resp.PositionX, resp.PositionY, resp.PositionAlignment, resp.Rotation, scale, scale, resp.CropTop, resp.CropBottom, resp.CropLeft, resp.CropRight, resp.Visible, resp.Locked, resp.BoundsType, resp.BoundsAlignment, resp.BoundsX, resp.BoundsY)
					req := obsws.NewSetSceneItemTransformRequest(scene, item, scale, scale, 0)
					_, err := req.SendReceive(c)
					if err != nil {
						log.Printf("Couldn't set transform of %s/%s: %v\n", scene, item, err)
						return
					}
				}
				time.Sleep(1 * time.Second)
			}
		}()
		time.Sleep(10 * time.Second)
	}
}

func main() {
	c, err := parseFlags()
	obsws.Logger.SetOutput(ioutil.Discard)
	obsws.SetReceiveTimeout(5 * time.Second)
	if err != nil {
		log.Println(err)
		os.Exit(2)
	}
	endpoints := strings.Split(c.endpoints, ",")
	for _, endpoint := range endpoints {
		go poll(endpoint, c.port, c.password, c.sceneName, c.itemName)
	}
	select {}
}
