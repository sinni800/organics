package organics

import(
	"sync"
	"time"
)

type host struct {
	access sync.RWMutex	
	requestHandlers map[interface{}]interface{}
	maxBufferSize int64
	pingRate, pingTimeout time.Duration
}

func newHost() host {
	h := host{}
	
	// Max message size: 1MB
	h.maxBufferSize = 1 * 1024 * 1024

	// Ping response every 5 minutes
	h.pingRate = 5 * time.Minute

	// Consider connection dead if no ping response in 30 seconds
	h.pingTimeout = 30 * time.Second
	
	return h
}

// Utility function to retrieve request handler for specified request name.
//
// Assumes server lock is not currently held.
func (s *host) getHandler(requestName interface{}) interface{} {
	s.access.RLock()
	defer s.access.RUnlock()
	return s.requestHandlers[requestName]
}

// SetMaxBufferSize sets the maximum size in bytes that the buffer which stores
// an single request may be.
//
// If an single JSON request exceeds this size, then the message will be
// refused, and the session killed.
//
// Default (1MB): 1 * 1024 * 1024
func (s *host) SetMaxBufferSize(size int64) {
	s.access.Lock()
	defer s.access.Unlock()

	s.maxBufferSize = size
}

// MaxBufferSize returns the maximum buffer size of this Server.
//
// See SetMaxBufferSize() for more information.
func (s *host) MaxBufferSize() int64 {
	s.access.RLock()
	defer s.access.RUnlock()

	return s.maxBufferSize
}

// SetPingTimeout specifies an duration which will be used to determine if an
// client is still considered connected.
//
// If an client leaves open it's long-polling POST request, then after
// PingRate() duration, the server will ask the client to respond ASAP, the
// client will then have PingTimeout() duration to respond, or else it will be
// considered disconnected, and the connection will be killed.
//
// This fixes an particular issue of leaving connection objects open forever,
// as web browsers are never required to close an HTTP connection (although
// most do), and some proxies might leave an connection open perminantly,
// causing the servers memory to fill with dead connections, and thus an crash
// occuring.
//
// Default (30 seconds): 30 * time.Second
func (s *host) SetPingTimeout(t time.Duration) {
	s.access.Lock()
	defer s.access.Unlock()

	s.pingTimeout = t
}

// PingTimeout returns the ping timeout.
//
// See SetPingTimeout() for more information.
func (s *host) PingTimeout() time.Duration {
	s.access.RLock()
	defer s.access.RUnlock()

	return s.pingTimeout
}

// SetPingRate specifies the interval duration at which the server should
// request long-polling clients to verify they are still connected and active.
//
// If an client leaves open it's long-polling POST request, then after
// PingRate() duration, the server will ask the client to respond ASAP, the
// client will then have PingTimeout() duration to respond, or else it will be
// considered disconnected, and the connection will be killed.
//
// This fixes an particular issue of leaving connection objects open forever,
// as web browsers are never required to close an HTTP connection (although
// most do), and some proxies might leave an connection open perminantly,
// causing the servers memory to fill with dead connections, and thus an crash
// occuring.
//
// Default (5 minutes): 5 * time.Minute
func (s *host) SetPingRate(t time.Duration) {
	s.access.Lock()
	defer s.access.Unlock()

	s.pingRate = t
}

// PingTimeout returns the ping rate of this server.
//
// See SetPingRate() for more information about this value.
func (s *host) PingRate() time.Duration {
	s.access.RLock()
	defer s.access.RUnlock()

	return s.pingRate
}