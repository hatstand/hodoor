package dash

import (
	"bytes"
	"context"
	"log"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

func isButtonPress(packet gopacket.Packet, mac []byte) bool {
	ethernetLayer := packet.Layer(layers.LayerTypeEthernet)
	ethernet := ethernetLayer.(*layers.Ethernet)
	if bytes.Equal(ethernet.SrcMAC, mac) {
		log.Println("Dash button pressed")
		return true
	}
	return false
}

func Listen(ctx context.Context, mac []byte) (<-chan interface{}, error) {
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
				if isButtonPress(packet, mac) {
					buttonCh <- nil
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return buttonCh, nil
}
