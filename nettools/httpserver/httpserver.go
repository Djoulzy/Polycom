package httpserver

import (
	"bytes"
	"html/template"
	"log"
	"time"

	// "github.com/gorilla/websocket"
	"github.com/fasthttp-contrib/websocket"
	"github.com/valyala/fasthttp"

	"github.com/Djoulzy/Polycom/hub"
	"github.com/Djoulzy/Polycom/monitoring"
	"github.com/Djoulzy/Polycom/urlcrypt"
	"github.com/Djoulzy/Tools/clog"
)

const (
	writeWait      = 5 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var (
	Newline = []byte{'\r', '\n'}
	Space   = []byte{' '}
)

type Manager struct {
	Httpaddr         string
	ServerName       string
	Hub              *hub.Hub
	ReadBufferSize   int
	WriteBufferSize  int
	HandshakeTimeout int
	CallToAction     func(*hub.Client, []byte)
	Cryptor          *urlcrypt.Cypher
}

// func (m *Manager) Connect() *websocket.Conn {
// 	u := url.URL{Scheme: "ws", Host: m.Httpaddr, Path: "/ws"}
// 	clog.Info("HTTPServer", "Connect", "Connecting to %s", u.String())
//
// 	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
// 	if err != nil {
// 		clog.Error("HTTPServer", "Connect", "%s", err)
// 		return nil
// 	}
//
// 	return conn
// }

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (m *Manager) Reader(conn *websocket.Conn, cli *hub.Client) {
	// conn := c.(*websocket.Conn)
	defer func() {
		conn.Close()
	}()

	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		// c.ReadProtect.Lock()
		conn.SetReadDeadline(time.Now().Add(pongWait))
		// c.ReadProtect.Unlock()
		clog.Debug("HTTPServer", "Reader", "PONG! from %s", cli.Name)
		return nil
	})
	for {
		// c.ReadProtect.Lock()
		// messType, message, err := conn.ReadMessage()
		// c.ReadProtect.Unlock()
		_, message, err := conn.ReadMessage()
		// clog.Debug("HTTPServer", "Writer", "Read from Client %s [%s]: %s", c.Name, c.ID, message)
		if err != nil {
			// clog.Error("HTTPServer", "Writer", "Type: %d, error: %v", messType, err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, Newline, Space, -1))
		// mess := Hub.NewMessage(c.CType, c, message)
		// c.Hub.Action <- mess
		go m.CallToAction(cli, message)
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (m *Manager) _write(ws *websocket.Conn, mt int, message []byte) error {
	ws.SetWriteDeadline(time.Now().Add(writeWait))
	return ws.WriteMessage(mt, message)
}

func (m *Manager) Writer(conn *websocket.Conn, cli *hub.Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case message, ok := <-cli.Send:
			if !ok {
				clog.Warn("HTTPServer", "Writer", "Error: %s", ok)
				cm := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Something went wrong !")
				if err := m._write(conn, websocket.CloseMessage, cm); err != nil {
					clog.Error("HTTPServer", "_close", "Cannot write CloseMessage to %s", cli.Name)
				}
				return
			}
			// clog.Debug("HTTPServer", "Writer", "Sending: %s", message)
			if err := m._write(conn, websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			clog.Debug("HTTPServer", "Writer", "Client %s Ping!", cli.Name)
			if err := m._write(conn, websocket.PingMessage, []byte{}); err != nil {
				return
			}
		case <-cli.Quit:
			cm := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "An other device is using your account !")
			if err := m._write(conn, websocket.CloseMessage, cm); err != nil {
				clog.Error("HTTPServer", "_close", "Cannot write CloseMessage to %s", cli.Name)
			}
			return
		}
	}
}

func (m *Manager) statusPage(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("text/html")
	handShake, _ := m.Cryptor.Encrypt_b64("MNTR|Monitoring|MNTR")
	var data = struct {
		Host   string
		Nb     int
		Users  map[string]*hub.Client
		Stats  string
		HShake string
	}{
		m.Httpaddr,
		len(m.Hub.Users),
		m.Hub.Users,
		monitoring.MachineLoad.String(),
		string(handShake),
	}

	homeTempl, err := template.ParseFiles("../html/status.html")
	if err != nil {
		clog.Error("HTTPServer", "statusPage", "%s", err)
		return
	}
	homeTempl.Execute(ctx, &data)
}

func (m *Manager) testPage(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("text/html")
	handShake, _ := m.Cryptor.Encrypt_b64("LOAD_1|TestPage|USER")

	var data = struct {
		Host   string
		HShake string
	}{
		m.Httpaddr,
		string(handShake),
	}

	homeTempl, err := template.ParseFiles("../html/client.html")
	if err != nil {
		clog.Error("HTTPServer", "testPage", "%s", err)
		return
	}
	homeTempl.Execute(ctx, &data)
}

func (m *Manager) WSHandler(c *websocket.Conn, headers *fasthttp.RequestHeader) {
	var ua string
	name := string(headers.Peek("Sec-Websocket-Key"))
	if len(headers.Peek("User-Agent")) > 0 {
		ua = string(headers.Peek("User-Agent"))
	} else {
		ua = "n/a"
	}

	if m.Hub.UserExists(name, hub.ClientUser) {
		clog.Warn("HTTPServer", "wsConnect", "Client %s already exists ... Refusing connection", name)
		return
	}

	clog.Test("HTTPServer", "WSHandler", "New Client")
	client := &hub.Client{Consistent: make(chan bool), Quit: make(chan bool),
		CType: hub.ClientUndefined, Send: make(chan []byte, 256), CallToAction: m.CallToAction,
		Name: name, Content_id: 0, Front_id: "", App_id: "", Country: "", User_agent: ua}
	client.Addr = c.RemoteAddr().String()

	m.Hub.Register <- client
	<-client.Consistent
	go m.Writer(c, client)
	m.Reader(c, client)
	m.Hub.Unregister <- client
	<-client.Consistent
}

// serveWs handles websocket requests from the peer.
func (m *Manager) wsConnect(ctx *fasthttp.RequestCtx) {
	var headers fasthttp.RequestHeader
	ctx.Request.Header.CopyTo(&headers)
	upgrader := websocket.New(func(c *websocket.Conn) { m.WSHandler(c, &headers) })
	err := upgrader.Upgrade(ctx)
	if err != nil {
		clog.Error("HTTPServer", "wsConnect", "%s", err)
		return
	}
}

func (m *Manager) Start(conf *Manager) {
	m = conf
	// m.Upgrader = upgrader.Upgrade{
	// 	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
	// 		return true
	// 	},
	// 	Error: func(ctx *fasthttp.RequestCtx, status int, reason error) {
	// 		clog.Error("httpserver", "Start", "Error %s", reason)
	// 	},
	// 	ReadBufferSize:   m.ReadBufferSize,
	// 	WriteBufferSize:  m.WriteBufferSize,
	// 	HandshakeTimeout: time.Duration(m.HandshakeTimeout) * time.Second,
	// } // use default options

	fs := fasthttp.FSHandler("../html/js", 0)
	h := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/test":
			m.testPage(ctx)
		case "/status":
			m.statusPage(ctx)
		case "/ws":
			m.wsConnect(ctx)
		case "/js":
			fs(ctx)
		default:
			ctx.Error("not found", fasthttp.StatusNotFound)
		}
	}

	err := fasthttp.ListenAndServe(m.Httpaddr, h)
	if err != nil {
		log.Fatal("HTTPServer: ", err)
	}
}
