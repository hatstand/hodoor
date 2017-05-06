package dash

import (
  "bytes"
  "fmt"
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
  fmt.Println("Src MAC: ", ethernet.SrcMAC)
  if bytes.Equal(ethernet.SrcMAC, DashMAC) {
    fmt.Println("Dash button pressed!")
    handler.HandleButtonPress()
  }
}

func Listen(handler Handler) error {
  handle, err := pcap.OpenLive("eth0", 1600, true, pcap.BlockForever)
  if err != nil {
    return err
  }
  defer handle.Close()
  err = handle.SetBPFFilter("arp")
  if err != nil {
    return err
  }
  packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
  go func() {
    for packet := range packetSource.Packets() {
      handlePacket(handler, packet)
    }
  }()
  return nil
}
