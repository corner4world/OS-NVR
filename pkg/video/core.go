package video

import (
	"context"
	"net/http"
	"nvr/pkg/log"
	"nvr/pkg/video/hls"
	"strconv"
	"sync"
)

// Server is an instance of rtsp-simple-server.
type Server struct {
	rtspAddress string
	hlsAddress  string
	pathManager *pathManager
	rtspServer  *rtspServer
	hlsServer   *hlsServer
	wg          *sync.WaitGroup
}

const readBufferCount = 2048

// NewServer allocates a server.
func NewServer(log *log.Logger, wg *sync.WaitGroup, rtspPort int, hlsPort int) *Server {
	// Only allow local connections.
	rtspAddress := "127.0.0.1:" + strconv.Itoa(rtspPort)
	hlsAddress := "127.0.0.1:" + strconv.Itoa(hlsPort)

	pathManager := newPathManager(wg, log)
	rtspServer := newRTSPServer(wg, rtspAddress, readBufferCount, pathManager, log)
	hlsServer := newHLSServer(wg, readBufferCount, pathManager, log)

	return &Server{
		rtspAddress: rtspAddress,
		hlsAddress:  hlsAddress,
		pathManager: pathManager,
		rtspServer:  rtspServer,
		hlsServer:   hlsServer,
		wg:          wg,
	}
}

// Start server.
func (s *Server) Start(ctx context.Context) error {
	ctx2, cancel := context.WithCancel(ctx)
	_ = cancel

	s.pathManager.start(ctx2)

	if err := s.rtspServer.start(ctx2); err != nil {
		cancel()
		return err
	}

	if err := s.hlsServer.start(ctx2, s.hlsAddress); err != nil {
		cancel()
		return err
	}
	return nil
}

// CancelFunc .
type CancelFunc func()

// HlsMuxerFunc .
type HlsMuxerFunc func() (IHLSMuxer, error)

// IHLSMuxer HLS muxer interface.
type IHLSMuxer interface {
	StreamInfo() (*hls.StreamInfo, error)
	WaitForSegFinalized()
	NextSegment(prevID uint64) (*hls.Segment, error)
}

// ServerPath .
type ServerPath struct {
	HlsAddress   string
	RtspAddress  string
	RtspProtocol string
	HLSMuxer     HlsMuxerFunc
}

// NewPath add path.
func (s *Server) NewPath(name string, newConf PathConf) (*ServerPath, CancelFunc, error) {
	hlsMuxer, err := s.pathManager.AddPath(name, newConf)
	if err != nil {
		return nil, nil, err
	}

	cancelFunc := func() {
		s.pathManager.RemovePath(name)
	}

	return &ServerPath{
		HlsAddress:   "http://" + s.hlsAddress + "/hls/" + name + "/index.m3u8",
		RtspAddress:  "rtsp://" + s.rtspAddress + "/" + name,
		RtspProtocol: "tcp",
		HLSMuxer:     hlsMuxer,
	}, cancelFunc, nil
}

// PathExist returns true if path exist.
func (s *Server) PathExist(name string) bool {
	return s.pathManager.pathExist(name)
}

// HandleHLS handle hls requests.
func (s *Server) HandleHLS() http.HandlerFunc {
	return s.hlsServer.HandleRequest()
}
