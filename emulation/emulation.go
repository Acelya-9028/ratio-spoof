package emulation

import (
	"embed"
	"encoding/json"
	generator2 "ratio-spoof/generator"
	"io"
	"io/fs"
	"sort"
	"strings"
)

type ClientInfo struct {
	Name   string `json:"name"`
	PeerID struct {
		Generator string `json:"generator"`
		Regex     string `json:"regex"`
	} `json:"peerId"`
	Key struct {
		Generator string `json:"generator"`
		Regex     string `json:"regex"`
	} `json:"key"`
	Rounding struct {
		Generator string `json:"generator"`
		Regex     string `json:"regex"`
	} `json:"rounding"`
	Query   string            `json:"query"`
	Headers map[string]string `json:"headers"`
}

type KeyGenerator interface {
	Key() string
}

type PeerIdGenerator interface {
	PeerId() string
}
type RoundingGenerator interface {
	Round(downloadCandidateNextAmount, uploadCandidateNextAmount, leftCandidateNextAmount, pieceSize int) (downloaded, uploaded, left int)
}

type Emulation struct {
	PeerIdGenerator
	KeyGenerator
	Query   string
	Name    string
	Headers map[string]string
	RoundingGenerator
}

func NewEmulation(code string) (*Emulation, error) {
	c, err := extractClient(code)
	if err != nil {
		return nil, err
	}

	peerG, err := generator2.NewRegexPeerIdGenerator(c.PeerID.Regex)
	if err != nil {
		return nil, err
	}

	keyG, err := generator2.NewDefaultKeyGenerator()
	if err != nil {
		return nil, err
	}

	roudingG, err := generator2.NewDefaultRoudingGenerator()
	if err != nil {
		return nil, err
	}

	return &Emulation{PeerIdGenerator: peerG, KeyGenerator: keyG, RoundingGenerator: roudingG,
		Headers: c.Headers, Name: c.Name, Query: c.Query}, nil

}

//go:embed static
var staticFiles embed.FS

// clientVersionMap contains all the available client versions
var clientVersionMap = initClientVersionMap()

// initClientVersionMap load client versions from static
func initClientVersionMap() map[string]struct{} {
	clientMap := make(map[string]struct{})
	
	fs.WalkDir(staticFiles, "static", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".json") {
			clientName := strings.TrimSuffix(d.Name(), ".json")
			clientMap[clientName] = struct{}{}
		}
		
		return nil
	})
	
	return clientMap
}

func extractClient(code string) (*ClientInfo, error) {

	f, err := staticFiles.Open("static/" + code + ".json")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bytes, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var client ClientInfo

	json.Unmarshal(bytes, &client)

	return &client, nil
}

// GetAvailableClients returns all a list of available clients
func GetAvailableClients() []string {
	clients := []string{}
	
	for client := range clientVersionMap {
		clients = append(clients, client)
	}
	
	sort.Strings(clients)
	
	return clients
}
