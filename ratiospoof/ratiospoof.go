package ratiospoof

import (
	"errors"
	"fmt"
	"ratio-spoof/bencode"
	"ratio-spoof/emulation"
	"ratio-spoof/input"
	"ratio-spoof/tracker"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gammazero/deque"
)

const (
	maxAnnounceHistory = 10
)

type RatioSpoof struct {
	TorrentInfo      *bencode.TorrentInfo
	Input            *input.InputParsed
	Tracker          *tracker.HttpTracker
	BitTorrentClient *emulation.Emulation
	AnnounceInterval int
	NumWant          int
	Seeders          int
	Leechers         int
	AnnounceCount    int
	Status           string
	AnnounceHistory  announceHistory
	Print            bool
	LastMessage      string
	SeedStartTime    time.Time
}

type AnnounceEntry struct {
	Count             int
	Downloaded        int
	PercentDownloaded float32
	Uploaded          int
	Left              int
}

type announceHistory struct {
	deque.Deque
}

func NewRatioSpoofState(input input.InputArgs) (*RatioSpoof, error) {
	dat, err := os.ReadFile(input.TorrentPath)
	if err != nil {
		return nil, err
	}

	client, err := emulation.NewEmulation(input.Client)
	if err != nil {
		return nil, errors.New("Error building the emulated client with the code")
	}

	torrentInfo, err := bencode.TorrentDictParse(dat)
	if err != nil {
		return nil, errors.New("failed to parse the torrent file")
	}

	httpTracker, err := tracker.NewHttpTracker(torrentInfo)
	if err != nil {
		return nil, err
	}

	inputParsed, err := input.ParseInput(torrentInfo)
	if err != nil {
		return nil, err
	}
	
	return &RatioSpoof{
		BitTorrentClient: client,
		TorrentInfo:      torrentInfo,
		Tracker:          httpTracker,
		Input:            inputParsed,
		NumWant:          200,
		Status:           "started",
		Print:            true,
		LastMessage:      "",
		SeedStartTime:    time.Now(),
	}, nil
}

func (a *announceHistory) pushValueHistory(value AnnounceEntry) {
	if a.Len() >= maxAnnounceHistory {
		a.PopFront()
	}
	a.PushBack(value)
}

func (r *RatioSpoof) gracefullyExit() {
	fmt.Printf("\nGracefully exiting...\n")
	r.Status = "stopped"
	r.NumWant = 0
	r.fireAnnounce(false)
	fmt.Printf("Gracefully exited successfully.\n")

}

func (r *RatioSpoof) Run() {
	sigCh := make(chan os.Signal)

	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	r.firstAnnounce()
	go func() {
		for {
			r.generateNextAnnounce()
			time.Sleep(time.Duration(r.AnnounceInterval) * time.Second)
			r.fireAnnounce(true)
		}
	}()
	<-sigCh
	r.Print = false
	r.gracefullyExit()
}

func (r *RatioSpoof) firstAnnounce() {
	r.addAnnounce(r.Input.InitialDownloaded, r.Input.InitialUploaded, calculateBytesLeft(r.Input.InitialDownloaded, r.TorrentInfo.TotalSize), (float32(r.Input.InitialDownloaded)/float32(r.TorrentInfo.TotalSize))*100)
	r.fireAnnounce(false)
}

func (r *RatioSpoof) updateSeedersAndLeechers(resp tracker.TrackerResponse) {
	r.Seeders = resp.Seeders
	r.Leechers = resp.Leechers
}

func (r *RatioSpoof) addAnnounce(currentDownloaded, currentUploaded, currentLeft int, percentDownloaded float32) {
	r.AnnounceCount++
	r.AnnounceHistory.pushValueHistory(AnnounceEntry{Count: r.AnnounceCount, Downloaded: currentDownloaded, Uploaded: currentUploaded, Left: currentLeft, PercentDownloaded: percentDownloaded})
}

func (r *RatioSpoof) fireAnnounce(retry bool) error {
	lastAnnounce := r.AnnounceHistory.Back().(AnnounceEntry)
	replacer := strings.NewReplacer("{infohash}", r.TorrentInfo.InfoHashURLEncoded,
		"{port}", fmt.Sprint(r.Input.Port),
		"{peerid}", r.BitTorrentClient.PeerId(),
		"{uploaded}", fmt.Sprint(lastAnnounce.Uploaded),
		"{downloaded}", fmt.Sprint(lastAnnounce.Downloaded),
		"{left}", fmt.Sprint(lastAnnounce.Left),
		"{key}", r.BitTorrentClient.Key(),
		"{event}", r.Status,
		"{numwant}", fmt.Sprint(r.NumWant))
	query := replacer.Replace(r.BitTorrentClient.Query)
	trackerResp, err := r.Tracker.Announce(query, r.BitTorrentClient.Headers, retry)
	if err != nil {
		log.Fatalf("failed to reach the tracker:\n%s ", err.Error())
	}

	if trackerResp != nil {
		r.updateSeedersAndLeechers(*trackerResp)
		r.AnnounceInterval = trackerResp.Interval
	}
	return nil
}

func (r *RatioSpoof) generateNextAnnounce() {
	lastAnnounce := r.AnnounceHistory.Back().(AnnounceEntry)
	currentDownloaded := lastAnnounce.Downloaded
	var downloadCandidate int

	if currentDownloaded < r.TorrentInfo.TotalSize {
		randomPiecesDownload := rand.Intn(10-1) + 1
		downloadCandidate = calculateNextTotalSizeByte(r.Input.DownloadSpeed, currentDownloaded, r.TorrentInfo.PieceSize, r.AnnounceInterval, r.TorrentInfo.TotalSize, randomPiecesDownload)
	} else {
		downloadCandidate = r.TorrentInfo.TotalSize
	}
	
	// Calculate base upload amount
	baseUpload := r.Input.UploadSpeed * r.AnnounceInterval
	
	// Calculate upload fluctuation based on multiple factors
	var fluctuation float64
	
	// Base fluctuation between 80% and 120% of base speed
	baseFluctuation := 0.8 + (rand.Float64() * 0.4)
	
	// Adjust based on number of leechers (more leechers = more upload opportunity)
	leecherFactor := 1.0
	if r.Leechers > 0 {
		// More leechers means more potential upload, but with diminishing returns
		leecherFactor = 1.0 + (float64(r.Leechers) / 100.0)
		if leecherFactor > 1.5 {
			leecherFactor = 1.5 // Cap the leecher bonus
		}
	} else if r.Input.WaitForLeechers {
		// If waiting for leechers, set upload to 0 and print warning
		leecherFactor = 0.0
		r.LastMessage = "[WARNING] No leechers detected. Waiting for leechers before continuing upload..."
	}
	
	// Combine all factors
	fluctuation = baseFluctuation * leecherFactor
	
	// Calculate final upload amount
	uploadCandidate := int(float64(baseUpload) * fluctuation)

	leftCandidate := calculateBytesLeft(downloadCandidate, r.TorrentInfo.TotalSize)

	d, u, l := r.BitTorrentClient.Round(downloadCandidate, uploadCandidate, leftCandidate, r.TorrentInfo.PieceSize)

	// Check if we just completed the download
	if d == r.TorrentInfo.TotalSize && lastAnnounce.Downloaded < r.TorrentInfo.TotalSize {
		r.Status = "completed"
	} else if r.Status == "started" || r.Status == "completed" {
		// After started/completed, use empty event for regular updates
		r.Status = ""
	}

	r.addAnnounce(d, u, l, (float32(d)/float32(r.TorrentInfo.TotalSize))*100)
}

func calculateNextTotalSizeByte(speedBytePerSecond, currentByte, pieceSizeByte, seconds, limitTotalBytes, randomPieces int) int {
	if speedBytePerSecond == 0 {
		return currentByte
	}
	totalCandidate := currentByte + (speedBytePerSecond * seconds)
	totalCandidate = totalCandidate + (pieceSizeByte * randomPieces)

	if limitTotalBytes != 0 && totalCandidate > limitTotalBytes {
		return limitTotalBytes
	}
	return totalCandidate
}

func calculateBytesLeft(currentBytes, totalBytes int) int {
	return totalBytes - currentBytes
}
