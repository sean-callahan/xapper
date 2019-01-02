package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sean-callahan/xapper"
	"github.com/tarm/serial"
)

func main() {
	addr := flag.String("http", "0.0.0.0:1776", "HTTP address")
	port := flag.String("p", "COM3", "Serial COM port")
	baud := flag.Int("b", 38400, "Serial baud rate")
	id := flag.Int("id", 0, "Device ID")
	heartbeat := flag.Int64("hb", 1000, "Heartbeat interval (ms)")
	verbose := flag.Bool("v", false, "Enable verbose logging")

	flag.Parse()

	if *id > 7 {
		log.Fatalln("invalid device id")
	}

	if *heartbeat <= 0 {
		log.Fatalln("cannot have a negative heartbeat interval")
	}

	c := &serial.Config{
		Name:        *port,
		Baud:        *baud,
		ReadTimeout: time.Second,
	}

	log.Println("Connecting to XAP on", c.Name)

	var logger *log.Logger
	if *verbose {
		logger = log.New(os.Stdout, "", log.LstdFlags)
	}

	d, err := xapper.NewDevice(*id, xapper.XAP800, c, logger)
	if err != nil {
		log.Fatalln("connection failed:", err)
	}
	defer d.Close()

	d.Start(time.Duration(*heartbeat) * time.Millisecond)

	r := xapper.Router()
	xapper.Global.Devices[0] = d

	if err := http.ListenAndServe(*addr, r); err != nil {
		log.Fatal(err)
	}
}
