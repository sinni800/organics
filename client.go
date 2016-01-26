package organics

import(
	"strings"
	"golang.org/x/net/websocket"
	"io"
	"fmt"
	"reflect"
	"net/http"
)

type Client struct {
	host
	url string
	tls bool
	origin string
	connected bool
	Connection *ClientConnection
	session *http.CookieJar
}

func NewClient(url string, tls bool, origin string) *Client {
	for _, val := range []string{"ws://", "wss://", "http://", "https://"} {
		url = strings.TrimPrefix(url, val)
	}
	
	c := new(Client)
	c.host = newHost()
	
	if tls {
		url = "wss://" + url
	} else {
		url = "ws://" + url
	}
	
	c.url = url	
	c.tls = tls	
	c.origin = origin
	
	return c
}

func (c *Client) Request(requestName interface{}, sequence ...interface{}) {
	c.Connection.Request(requestName, sequence...)
}

func (c *Client) Connect() error {
	ws, err := websocket.Dial(c.url, "", c.origin)

	if err != nil {
		println("err in ws: " + err.Error() + " arg " + c.url + " " + c.origin)
		return err
	}

	connection := newClientConnection(c.url, WebSocket)

	go c.webSocketWrite(ws, connection.connection)
	go c.webSocketWaitForDeath(ws, connection.connection)
	connection.disconnectTimer(c.PingTimeout(), c.PingRate())

	//c.doConnectHandler(connection)
	
	c.Connection = connection
	
	go func() {
		for {
			msg, err := receiveMessage(ws, c.MaxBufferSize())
			if err != nil {
				if err != io.EOF {
					logger().Println("receiveMessage() failed:", err)
				}
				connection.Kill()
				c.Connect()
				break
			}
	
			defer func() {
				if e := recover(); e != nil {
					logger().Println(fmt.Sprint(e))
				}
			}()
			c.webSocketHandleMessage(msg, ws, connection.connection)
		}
	}()
	return nil
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