package GameSpy

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/HeroesAwaken/GoAwaken/core"
	"github.com/HeroesAwaken/GoFesl/log"
)

type Client struct {
	name       string
	conn       *net.Conn
	recvBuffer []byte
	eventChan  chan ClientEvent
	IsActive   bool
	reader     *bufio.Reader
	RedisState *core.RedisState
	IpAddr     net.Addr
	State      ClientState
	FESL       bool
}

type ClientState struct {
	GameName        string
	ServerChallenge string
	ClientChallenge string
	ClientResponse  string
	BattlelogID     int
	Username        string
	PlyName         string
	PlyEmail        string
	PlyCountry      string
	PlyPid          int
	Sessionkey      int
	Confirmed       bool
	Banned          bool
	IpAddress       net.Addr
	HasLogin        bool
	ProfileSent     bool
	LoggedOut       bool
	HeartTicker     *time.Ticker
}

// ClientEvent is the generic struct for events
// by this Client
type ClientEvent struct {
	Name string
	Data interface{}
}

// New creates a new Client and starts up the handling of the connection
func (client *Client) New(name string, conn *net.Conn) (chan ClientEvent, error) {
	client.name = name
	client.conn = conn
	client.IpAddr = (*client.conn).RemoteAddr()
	client.eventChan = make(chan ClientEvent, 1000)
	client.reader = bufio.NewReader(*client.conn)
	client.IsActive = true

	go client.handleRequest()

	return client.eventChan, nil
}

func (client *Client) Write(command string) error {
	if !client.IsActive {
		log.Notef("%s: Trying to write to inactive client.\n%v", client.name, command)
		return errors.New("client is not active. Can't send message")
	}

	log.Debugln("Write message:", command)

	(*client.conn).Write([]byte(command))
	return nil
}

// WriteError Handy for informing the user they're a piece of shit.
func (client *Client) WriteError(code string, message string) error {
	err := client.Write("\\error\\\\err\\" + code + "\\fatal\\\\errmsg\\" + message + "\\id\\1\\final\\")
	return err
}

func (client *Client) processCommand(command string) {
	gsPacket, err := ProcessCommand(command)
	if err != nil {
		log.Errorf("%s: Error processing command %s.\n%v", client.name, command, err)
		client.eventChan <- ClientEvent{
			Name: "error",
			Data: err,
		}
		return
	}

	client.eventChan <- ClientEvent{
		Name: "command." + gsPacket.Query,
		Data: gsPacket,
	}
	client.eventChan <- ClientEvent{
		Name: "command",
		Data: gsPacket,
	}
}

func (client *Client) Close() {
	log.Notef("%s: Client closing connection.", client.name)
	client.eventChan <- ClientEvent{
		Name: "close",
		Data: client,
	}
	client.IsActive = false
}

func (client *Client) WriteFESL(msgType string, msg map[string]string, msgType2 uint32) error {

	if !client.IsActive {
		log.Notef("%s: Trying to write to inactive Client.\n%v", client.name, msg)
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

	n, err := (*client.conn).Write(buf.Bytes())
	if err != nil {
		fmt.Println("Writing failed:", n, err)
	}
	return nil
}

func (client *Client) readFESL(data []byte) []byte {
	p := bytes.NewBuffer(data)
	i := 0
	log.Debugln(hex.EncodeToString(data))
	var payloadRaw []byte
	for {
		// Create a copy at this point in case we have to abort later
		// And send back the packet to get the rest
		curData := p
		outCommand := new(CommandFESL)

		var payloadID uint32
		var payloadLen uint32

		payloadTypeRaw := make([]byte, 4)
		_, err := p.Read(payloadTypeRaw)
		if err != nil {
			return nil
		}

		payloadType := string(payloadTypeRaw)

		binary.Read(p, binary.BigEndian, &payloadID)

		if p.Len() < 4 {
			log.Noteln("Strange anomly")
			return nil
		}

		binary.Read(p, binary.BigEndian, &payloadLen)

		log.Noteln("Current message: " + payloadType + " - " + fmt.Sprint(payloadID) + " - " + fmt.Sprint(payloadLen))

		if (payloadLen - 12) > uint32(len(p.Bytes())) {
			log.Noteln("Packet not fully read")
			return curData.Bytes()
		}

		payloadRaw = make([]byte, (payloadLen - 12))
		p.Read(payloadRaw)

		payload := ProcessFESL(string(payloadRaw))

		outCommand.Query = payloadType
		outCommand.PayloadID = payloadID
		outCommand.Message = payload

		client.eventChan <- ClientEvent{
			Name: "command." + payloadType,
			Data: outCommand,
		}
		client.eventChan <- ClientEvent{
			Name: "command",
			Data: outCommand,
		}

		i++
	}

	return nil
}

func (client *Client) handleRequest() {
	client.IsActive = true
	buf := make([]byte, 16384) // buffer
	tempBuf := []byte{}

	for client.IsActive {
		n, err := (*client.conn).Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Debugf("%s: Reading from client threw an error. %v", client.name, err)
				client.eventChan <- ClientEvent{
					Name: "error",
					Data: err,
				}
				client.eventChan <- ClientEvent{
					Name: "close",
					Data: client,
				}
				return
			}
			// If we receive an EndOfFile, close this function/goroutine
			log.Notef("%s: Client closing connection.", client.name)
			client.eventChan <- ClientEvent{
				Name: "close",
				Data: client,
			}
			return

		}

		if client.FESL {
			if tempBuf != nil {
				tempBuf = append(tempBuf, buf[:n]...)
				tempBuf = client.readFESL(buf[:n])
			} else {
				tempBuf = client.readFESL(buf[:n])
			}
			buf = make([]byte, 16384) // new fresh buffer
			continue
		}

		client.recvBuffer = append(client.recvBuffer, buf[:n]...)

		message := strings.TrimSpace(string(client.recvBuffer))

		log.Debugln("Got message:", hex.EncodeToString(client.recvBuffer))

		if strings.Index(message, "\\final\\") == -1 {
			if len(client.recvBuffer) > 1024 {
				// We don't support more than 2048 long messages
				client.recvBuffer = make([]byte, 0)
			}
			continue
		}

		client.eventChan <- ClientEvent{
			Name: "data",
			Data: message,
		}

		commands := strings.Split(message, "\\final\\")
		for _, command := range commands {
			if len(command) == 0 {
				continue
			}

			client.processCommand(command)
		}

		// Add unprocessed commands back into recvBuffer
		client.recvBuffer = []byte(commands[(len(commands) - 1)])
	}

}
