package webtail

import (
	"net/http"

	"github.com/LeKovr/go-base/log"
)

// Config defines local application flags
type Config struct {
	Root        string `long:"root"  default:"log/"  description:"Root directory for log files"`
	Bytes       int64  `long:"bytes" default:"5000"  description:"tail from the last Nth location"`
	Lines       int    `long:"lines" default:"100"   description:"keep N old lines for new consumers"`
	MaxLineSize int    `long:"split" default:"180"   description:"split line if longer"`
	ListCache   int    `long:"cache" default:"2"      description:"Time to cache file listing (sec)"`
	Poll        bool   `long:"poll"  description:"use polling, instead of inotify"`
	Trace       bool   `long:"trace" description:"trace worker channels"`

	ClientBufferSize  int `long:"out_buf"      default:"256"  description:"Client Buffer Size"`
	WSReadBufferSize  int `long:"ws_read_buf"  default:"1024" description:"WS Read Buffer Size"`
	WSWriteBufferSize int `long:"ws_write_buf" default:"1024" description:"WS Write Buffer Size"`
}

// Service holds WebTail service
type Service struct {
	cfg *Config
	hub *ClientHub
	lg  log.Logger
}

// New creates WebTail service
func New(lg log.Logger, cfg *Config) (*Service, error) {
	tail, err := NewTailHub(lg, cfg)
	if err != nil {
		return nil, err
	}
	hub := newClientHub(lg, tail)
	return &Service{cfg: cfg, hub: hub, lg: lg}, nil
}

// Run runs a message hub
func (wt *Service) Run() {
	wt.hub.run()
}

// Handle handles websocket requests from the peer
func (wt *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wsUpgrader := upgrader(wt.cfg.WSReadBufferSize, wt.cfg.WSWriteBufferSize)
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		wt.lg.Println(err)
		return
	}
	client := &Client{hub: wt.hub, conn: conn, send: make(chan []byte, wt.cfg.ClientBufferSize)}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
