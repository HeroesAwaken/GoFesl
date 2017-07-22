package lib

// Needed since we are using this for opening the connection

//Packet ...
type Packet struct {
	packet map[string]string
}

//NewPacket returns a new packet object
func NewPacket() *Packet {
	return &Packet{}
}

//AddField ...
func (p *Packet) AddField(key, value string) *Packet {
	p.packet[key] = value
	return p
}

//Raw ...
func (p *Packet) Raw() map[string]string {
	return p.packet
}
