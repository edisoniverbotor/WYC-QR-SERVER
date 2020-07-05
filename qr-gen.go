package main

// MIT Licensed - see LICENSE

import (
	"fmt"
	"os"
	"strings"

	"github.com/pschlump/goqrcode"
)

// GenQR generates the image for the QR code as a .png in basePath.
// fn is the final path of the QR.
func GenQR(QRDir, QRUri, HostPort, id string) (uri, pth string, err error) {
	pth = fmt.Sprintf("./%s/%s.png", QRDir, id)
	pth = strings.Replace(pth, "/./", "/", -1)
	uri = fmt.Sprintf("http://%s/Q/%s", gCfg.HostPort, id)
	uri = strings.Replace(uri, "/./", "/", -1)

	if dbFlag["GenQR"] {
		fmt.Printf("GenQR: pth [%s] uri [%s]\n", pth, uri)
	}

	redundancy := goqrcode.Highest
	switch gCfg.Level {
	case "h", "high", "H":
	case "m", "medium", "M":
		redundancy = goqrcode.Medium
	case "l", "low", "L":
		redundancy = goqrcode.Low
	default:
		err = fmt.Errorf("Invalid level")
		return
	}

	// Generate the QR code in internal format
	var q *goqrcode.QRCode
	q, err = goqrcode.New(uri, redundancy)
	goqrcode.CheckError(err)

	// Output QR Code as a PNG
	var png []byte
	png, err = q.PNG(gCfg.QRSize)
	if err != nil {
		err = fmt.Errorf("Failed to generate QR: %s", err)
		return
	}
	goqrcode.CheckError(err)

	var fh *os.File
	fh, err = Fopen(pth, "w")
	if err != nil {
		err = fmt.Errorf("Failed to write QR: %s", err)
		return
	}
	defer fh.Close()
	fh.Write(png)

	return
}

/* vim: set noai ts=4 sw=4: */
