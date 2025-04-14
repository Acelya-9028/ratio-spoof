package input

import (
	"errors"
	"fmt"
	"ratio-spoof/bencode"
	"strconv"
	"strings"
)

const (
	minPortNumber     = 1
	maxPortNumber     = 65535
	speedSuffixLength = 4
)

type InputArgs struct {
	Client             string
	Debug              bool
	DownloadSpeed      string
	InitialDownloaded  string
	InitialUploaded    string
	Port               int
	TorrentPath        string
	UploadSpeed        string
	WaitForLeechers    bool
}

type InputParsed struct {
	Debug              bool
	DownloadSpeed      int
	InitialDownloaded  int
	InitialUploaded    int
	Port               int
	TorrentPath        string
	UploadSpeed        int
	WaitForLeechers    bool
}

var validSpeedSufixes = [...]string{"kbps", "mbps"}

func (i *InputArgs) ParseInput(torrentInfo *bencode.TorrentInfo) (*InputParsed, error) {
	downloaded, err := extractInputInitialByteCount(i.InitialDownloaded, torrentInfo.TotalSize, true)
	if err != nil {
		return nil, err
	}
	uploaded, err := extractInputInitialByteCount(i.InitialUploaded, torrentInfo.TotalSize, false)
	if err != nil {
		return nil, err
	}
	downloadSpeed, err := extractInputByteSpeed(i.DownloadSpeed)
	if err != nil {
		return nil, err
	}
	uploadSpeed, err := extractInputByteSpeed(i.UploadSpeed)
	if err != nil {
		return nil, err
	}

	if i.Port < minPortNumber || i.Port > maxPortNumber {
		return nil, errors.New(fmt.Sprint("port number must be between %i and %i", minPortNumber, maxPortNumber))
	}

	return &InputParsed{
		Debug:             i.Debug,
		DownloadSpeed:     downloadSpeed,
		InitialDownloaded: downloaded,
		InitialUploaded:   uploaded,
		Port:              i.Port,
		TorrentPath:       i.TorrentPath,
		UploadSpeed:       uploadSpeed,
		WaitForLeechers:   i.WaitForLeechers,
	}, nil
}

func checkSpeedSufix(input string) (valid bool, suffix string) {
	for _, v := range validSpeedSufixes {
		if strings.HasSuffix(strings.ToLower(input), v) {
			return true, input[len(input)-4:]
		}
	}
	return false, ""
}

func extractInputInitialByteCount(initialSizeInput string, totalBytes int, errorIfHigher bool) (int, error) {
	if !strings.HasSuffix(initialSizeInput, "%") {
		return 0, errors.New("initial value must be in percentage")
	}
	
	percent, err := strconv.ParseFloat(initialSizeInput[:len(initialSizeInput)-1], 64)
	if err != nil {
		return 0, errors.New("invalid percentage value")
	}
	
	if percent < 0 || percent > 100 {
		return 0, errors.New("percentage must be between 0 and 100")
	}
	
	byteCount := int(float64(totalBytes) * percent / 100)
	
	if errorIfHigher && byteCount > totalBytes {
		return 0, errors.New("initial downloaded can not be higher than the torrent size")
	}
	if byteCount < 0 {
		return 0, errors.New("initial value can not be negative")
	}
	return byteCount, nil
}

// Takes an dirty speed input and returns the bytes per second based on the suffixes
// example 1kbps(string) > 1024 bytes per second (int)
func extractInputByteSpeed(initialSpeedInput string) (int, error) {
	ok, suffix := checkSpeedSufix(initialSpeedInput)
	if !ok {
		return 0, fmt.Errorf("speed must be in %v", validSpeedSufixes)
	}
	speedVal, err := strconv.ParseFloat(initialSpeedInput[:len(initialSpeedInput)-speedSuffixLength], 64)
	if err != nil {
		return 0, errors.New("invalid speed number")
	}
	if speedVal < 0 {
		return 0, errors.New("speed can not be negative")
	}

	if suffix == "kbps" {
		speedVal *= 1024
	} else {
		speedVal = speedVal * 1024 * 1024
	}
	ret := int(speedVal)
	return ret, nil
}
