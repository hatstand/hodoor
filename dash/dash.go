package dash

import (
	"bytes"
	"log"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

var DashMAC = mustParseMAC("68:37:e9:99:de:58")

func mustParseMAC(s string) net.HardwareAddr {
	mac, err := net.ParseMAC(s)
	if err != nil {
		panic(err)
	}
	return mac
}

type Handler interface {
	HandleButtonPress()
}

func handlePacket(handler Handler, packet gopacket.Packet) {
	ethernetLayer := packet.Layer(layers.LayerTypeEthernet)
	ethernet := ethernetLayer.(*layers.Ethernet)
	if bytes.Equal(ethernet.SrcMAC, DashMAC) {
		log.Println("Dash button pressed")
		handler.HandleButtonPress()
	}
}

func Listen(handler Handler) error {
	handle, err := pcap.OpenLive("wlan0", 1600, true, pcap.BlockForever)
	if err != nil {
		return err
	}
	err = handle.SetBPFFilter("arp")
	if err != nil {
		return err
	}
	defer handle.Close()
	log.Println("Listening for ARP packets")
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		handlePacket(handler, packet)
	}
	return nil
}
