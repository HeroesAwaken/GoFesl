package GameSpy

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"encoding/binary"
	"encoding/hex"

	"github.com/HeroesAwaken/GoAwaken/core"
	"github.com/SpencerSharkey/GoFesl/log"
)

type ClientTLS struct {
	name       string
	conn       *tls.Conn
	recvBuffer []byte
	eventChan  chan ClientTLSEvent
	IsActive   bool
	IpAddr     net.Addr
	RedisState *core.RedisState
	State      ClientTLSState
	FESL       bool
}

type ClientTLSState struct {
	GameName           string
	ServerChallenge    string
	ClientTLSChallenge string
	ClientTLSResponse  string
	BattlelogID        int
	Username           string
	PlyName            string
	PlyEmail           string
	PlyCountry         string
	PlyPid             int
	Sessionkey         int
	Confirmed          bool
	Banned             bool
	IpAddress          net.Addr
	HasLogin           bool
	ProfileSent        bool
	LoggedOut          bool
	HeartTicker        *time.Ticker
}

type CommandFESL struct {
	Message   map[string]string
	Query     string
	PayloadID uint32
}

// ClientTLSEvent is the generic struct for events
// by this ClientTLS
type ClientTLSEvent struct {
	Name string
	Data interface{}
}

// New creates a new ClientTLS and starts up the handling of the connection
func (clientTLS *ClientTLS) New(name string, conn *tls.Conn) (chan ClientTLSEvent, error) {
	clientTLS.name = name
	clientTLS.conn = conn
	clientTLS.IpAddr = (*clientTLS.conn).RemoteAddr()
	clientTLS.eventChan = make(chan ClientTLSEvent, 1000)
	clientTLS.IsActive = true

	go clientTLS.handleRequest()

	return clientTLS.eventChan, nil
}

func (clientTLS *ClientTLS) WriteFESL(msgType string, msg map[string]string, msgType2 uint32) error {

	if !clientTLS.IsActive {
		log.Notef("%s: Trying to write to inactive ClientTLS.\n%v", clientTLS.name, msg)
		return errors.New("ClientTLS is not active. Can't send message")
	}
	var lena int32
	var buf bytes.Buffer

	payloadEncoded := SerializeFESL(msg)
	baselen := len(payloadEncoded)
	lena = int32(baselen + 12)

	buf.Write([]byte(msgType))

	err := binary.Write(&buf, binary.BigEndian, &msgType2)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}

	err = binary.Write(&buf, binary.BigEndian, &lena)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}

	buf.Write([]byte(payloadEncoded))

	log.Debugln("Write message:", msg, msgType, msgType2)

	n, err := (*clientTLS.conn).Write(buf.Bytes())
	if err != nil {
		fmt.Println("Writing failed:", n, err)
	}
	return nil
}

func (clientTLS *ClientTLS) readFESL(data []byte) {
	p := bytes.NewBuffer(data)
	i := 0
	log.Debugln(hex.EncodeToString(data))
	var payloadRaw []byte
	for {
		outCommand := new(CommandFESL)

		var payloadId uint32
		var payloadLen uint32

		payloadTypeRaw := make([]byte, 4)
		_, err := p.Read(payloadTypeRaw)
		if err != nil {
			return
		}

		payloadType := string(payloadTypeRaw)

		binary.Read(p, binary.BigEndian, &payloadId)

		if p.Len() < 4 {
			log.Noteln("Strange anomly")
			return
		}

		binary.Read(p, binary.BigEndian, &payloadLen)

		log.Noteln("Current message: " + payloadType + " - " + fmt.Sprint(payloadId) + " - " + fmt.Sprint(payloadLen))

		if payloadLen < 12 || payloadLen > 1024 {
			log.Noteln("Strange anomly, would wrap value")
			return
		}

		payloadRaw = make([]byte, (payloadLen - 12))
		p.Read(payloadRaw)

		payload := ProcessFESL(string(payloadRaw))

		outCommand.Query = payloadType
		outCommand.PayloadID = payloadId
		outCommand.Message = payload

		clientTLS.eventChan <- ClientTLSEvent{
			Name: "command." + payloadType,
			Data: outCommand,
		}
		clientTLS.eventChan <- ClientTLSEvent{
			Name: "command",
			Data: outCommand,
		}

		i++
	}
}

func (clientTLS *ClientTLS) Close() {
	log.Notef("%s: ClientTLS closing connection.", clientTLS.name)
	clientTLS.eventChan <- ClientTLSEvent{
		Name: "close",
		Data: clientTLS,
	}
	clientTLS.IsActive = false
}

func (clientTLS *ClientTLS) handleRequest() {
	clientTLS.IsActive = true
	buf := make([]byte, 8192) // buffer

	for clientTLS.IsActive {
		n, err := (*clientTLS.conn).Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Debugf("%s: Reading from ClientTLS threw an error. %v", clientTLS.name, err)
				clientTLS.eventChan <- ClientTLSEvent{
					Name: "error",
					Data: err,
				}
				clientTLS.eventChan <- ClientTLSEvent{
					Name: "close",
					Data: clientTLS,
				}
				return
			}
			// If we receive an EndOfFile, close this function/goroutine
			log.Notef("%s: ClientTLS closing connection.", clientTLS.name)
			clientTLS.eventChan <- ClientTLSEvent{
				Name: "close",
				Data: clientTLS,
			}
			return

		}
		clientTLS.readFESL(buf[:n])
		buf = make([]byte, 8192) // buffer
	}

}
