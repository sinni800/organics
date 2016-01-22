package organics

import(
	"strings"
	"golang.org/x/net/websocket"
	"io"
	"fmt"
	"reflect"
)

type Client struct {
	host
	url string
	tls bool
	origin string
	connected bool
	Connection *ClientConnection
}

func NewClient(url string, tls bool, origin string) *Client {
	for _, val := range []string{"ws://", "wss://", "http://", "https://"} {
		url = strings.TrimPrefix(url, val)
	}
	
	c := new(Client)
	
	if tls {
		url = "wss://" + url
	} else {
		url = "ws://" + url
	}
	
	c.url = url	
	c.tls = tls	
	
	return c
}

func (c *Client) Connect() error {
	ws, err := websocket.Dial(c.url, "", c.origin)

	if err != nil {
		return err
	}

	connection := newClientConnection(ws.Request().RemoteAddr, WebSocket)

	go c.webSocketWrite(ws, connection.connection)
	go c.webSocketWaitForDeath(ws, connection.connection)
	connection.disconnectTimer(c.PingTimeout(), c.PingRate())

	c.doConnectHandler(connection)

	for {
		msg, err := receiveMessage(ws, c.MaxBufferSize())
		if err != nil {
			if err != io.EOF {
				logger().Println("receiveMessage() failed:", err)
			}
			connection.Kill()
			break
		}

		defer func() {
			if e := recover(); e != nil {
				logger().Println(fmt.Sprint(e))
			}
		}()
		s.webSocketHandleMessage(msg, ws, connection)
	}
}

// Handle defines that when an request with the specified requestName comes in,
// that the requestHandler function will be invoked in order to handle the
// request.
//
// The requestName parameter may be of any valid json.Marshal() type.
//
// The requestHandler parameter must be an function, with the type specified
// below, where T is any valid json.Marshal() type.
//
// func(T, T, ..., *Session) (T, T, ...)
func (s *Client) Handle(requestName, requestHandler interface{}) {
	s.access.Lock()
	defer s.access.Unlock()

	fn := reflect.ValueOf(requestHandler)
	if fn.Kind() != reflect.Func {
		panic("requestHandler parameter type incorrect! Must be function!")
	}

	fnType := fn.Type()
	connectionParam := fnType.In(fnType.NumIn() - 1)
	var connectionType *ClientConnection
	if connectionParam != reflect.TypeOf(connectionType) {
		panic("requestHandler parameter type incorrect! Last parameter must be *organics.ClientConnection")
	}

	if requestHandler == nil {
		delete(s.requestHandlers, requestName)
	} else {
		s.requestHandlers[requestName] = requestHandler
	}
}