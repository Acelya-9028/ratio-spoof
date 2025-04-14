package main

import (
	"flag"
	"fmt"
	"ratio-spoof/input"
	"ratio-spoof/printer"
	"ratio-spoof/ratiospoof"
	"log"
	"os"
	"strings"
)

func main() {
	//required
	torrentPath := flag.String("t", "", "torrent path")

	//optional
	download := flag.String("d", "100%:0kbps", "initial downloaded percentage and download speed (format: <percentage>:<speed>)")
	upload := flag.String("u", "0%:0kbps", "initial uploaded percentage and upload speed (format: <percentage>:<speed>)")
	client := flag.String("c", "qbit-5.0.4", "emulated client")
	port := flag.Int("p", 8999, "a PORT")
	debug := flag.Bool("debug", false, "")
	waitForLeechers := flag.Bool("wait-leechers", false, "wait for leechers instead of continuing with reduced speed")

	flag.Usage = func() {
		fmt.Printf("usage: %s -t <TORRENT_PATH> -d <INITIAL_DOWNLOADED>:<DOWNLOAD_SPEED> -u <INITIAL_UPLOADED>:<UPLOAD_SPEED>\n", os.Args[0])
		fmt.Print(`
optional arguments:
	-h					show this help message and exit
	-p [PORT]			change the port number, default: 8999
	-c [CLIENT_CODE]	the client emulation, default: qbit-5.0.4
	-wait-leechers		wait for leechers instead of uploading with normal speed
	  
required arguments:
	-t  <TORRENT_PATH>     
	-d  <INITIAL_DOWNLOADED>:<DOWNLOAD_SPEED> 
	-u  <INITIAL_UPLOADED>:<UPLOAD_SPEED> 
	  
<INITIAL_DOWNLOADED> and <INITIAL_UPLOADED> must be in %
<DOWNLOAD_SPEED> and <UPLOAD_SPEED> must be in kbps or mbps
[CLIENT_CODE] options: qbit-4.0.3, qbit-4.3.9, qbit-4.6.5, qbit-5.0.4
`)
	}

	flag.Parse()

	if *torrentPath == "" {
		flag.Usage()
		return
	}

	// Parse download and upload parameters
	initialDownloaded, downloadSpeed, err := parseCombinedParameter(*download)
	if err != nil {
		log.Fatalf("Error parsing download parameter: %v", err)
	}

	initialUploaded, uploadSpeed, err := parseCombinedParameter(*upload)
	if err != nil {
		log.Fatalf("Error parsing upload parameter: %v", err)
	}

	r, err := ratiospoof.NewRatioSpoofState(
		input.InputArgs{
			TorrentPath:       *torrentPath,
			InitialDownloaded: initialDownloaded,
			DownloadSpeed:     downloadSpeed,
			InitialUploaded:   initialUploaded,
			UploadSpeed:       uploadSpeed,
			Port:              *port,
			Debug:             *debug,
			Client:            *client,
			WaitForLeechers:   *waitForLeechers,
		})

	if err != nil {
		log.Fatalln(err)
	}

	go printer.PrintState(r)
	r.Run()
}

func parseCombinedParameter(param string) (string, string, error) {
	parts := strings.Split(param, ":")
	if len(parts) == 1 {
		// If only speed is provided, use default percentage
		if strings.HasSuffix(parts[0], "kbps") || strings.HasSuffix(parts[0], "mbps") {
			if strings.HasPrefix(param, "d") {
				return "100%", parts[0], nil
			}
			return "0%", parts[0], nil
		}
		return "", "", fmt.Errorf("invalid parameter format: %s", param)
	}
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid parameter format: %s", param)
	}
	return parts[0], parts[1], nil
}
