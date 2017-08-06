package dash

import (
	"bytes"
	"context"
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

func isButtonPress(packet gopacket.Packet) bool {
	ethernetLayer := packet.Layer(layers.LayerTypeEthernet)
	ethernet := ethernetLayer.(*layers.Ethernet)
	if bytes.Equal(ethernet.SrcMAC, DashMAC) {
		log.Println("Dash button pressed")
		return true
	}
	return false
}

func Listen(ctx context.Context) (<-chan interface{}, error) {
	handle, err := pcap.OpenLive("wlan0", 1600, true, pcap.BlockForever)
	if err != nil {
		return nil, err
	}
	err = handle.SetBPFFilter("arp")
	if err != nil {
		return nil, err
	}

	log.Println("Listening for ARP packets")
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	buttonCh := make(chan interface{})
	go func() {
		defer close(buttonCh)
		defer handle.Close()
		for {
			select {
			case packet := <-packetSource.Packets():
				if isButtonPress(packet) {
					buttonCh <- nil
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return buttonCh, nil
}
