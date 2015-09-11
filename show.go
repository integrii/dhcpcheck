package main

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"./dhcp"
	"github.com/cmatsuoka/dncomp"
)

type option struct {
	Len  int
	Name string
}

var options map[byte]option
var messageType map[byte]string

func init() {
	options = map[byte]option{
		dhcp.PadOption:             {0, "Pad Option"},
		dhcp.Router:                {-1, "Router"},
		dhcp.SubnetMask:            {4, "Subnet Mask"},
		dhcp.DomainNameServer:      {-1, "Domain Name Server"},
		dhcp.HostName:              {-1, "Host Name"},
		dhcp.DomainName:            {-1, "Domain Name"},
		dhcp.BroadcastAddress:      {4, "Broadcast Address"},
		dhcp.StaticRoute:           {-1, "Static Route"},
		dhcp.IPAddressLeaseTime:    {4, "IP Address Lease Time"},
		dhcp.DHCPMessageType:       {1, "DHCP Message Type"},
		dhcp.ServerIdentifier:      {4, "Server Identifier"},
		dhcp.RenewalTimeValue:      {4, "Renewal Time Value"},
		dhcp.RebindingTimeValue:    {4, "Rebinding Time Value"},
		dhcp.VendorSpecific:        {-1, "Vendor Specific"},
		dhcp.NetBIOSNameServer:     {-1, "NetBIOS Name Server"},
		dhcp.RequestedIPAddress:    {-1, "Requested IP Address"},
		dhcp.VendorClassIdentifier: {-1, "Vendor Class Identifier"},
		dhcp.MaxDHCPMessageSize:    {2, "Max DHCP Message Size"},
		dhcp.ParameterRequestList:  {-1, "Parameter Request List"},
		dhcp.ClientIdentifier:      {-1, "Client Identifier"},
		dhcp.DomainSearch:          {-1, "Domain Search"},
		dhcp.ClientFQDN:            {-1, "Client FQDN"},
		dhcp.WebProxyServer:        {-1, "Web Proxy Server"},
	}

	messageType = map[byte]string{
		dhcp.DHCPDiscover: "DHCPDISCOVER",
		dhcp.DHCPOffer:    "DHCPOFFER",
		dhcp.DHCPRequest:  "DHCPREQUEST",
		dhcp.DHCPDecline:  "DHCPDECLINE",
		dhcp.DHCPAck:      "DHCPACK",
		dhcp.DHCPNack:     "DHCPNACK",
		dhcp.DHCPRelease:  "DHCPRELEASE",
		dhcp.DHCPInform:   "DHCPINFORM",
	}
}

func wireString(b []byte) string {
	var buf bytes.Buffer
	i := 0
	for {
		length := int(b[i])
		if length == 0 {
			break
		}
		length += i
		if length > len(b) {
			break
		}
		buf.Write(b[i:length])
		buf.WriteString(".")

		i += 1 + length
		if i >= len(b) {
			break
		}
	}
	return buf.String()
}

func showOptions(p *dhcp.Packet) {
	b16 := func(data []byte) uint16 {
		buf := bytes.NewBuffer(data)
		var x uint16
		binary.Read(buf, binary.BigEndian, &x)
		return x
	}

	b32 := func(data []byte) uint32 {
		buf := bytes.NewBuffer(data)
		var x uint32
		binary.Read(buf, binary.BigEndian, &x)
		return x
	}

	ip4 := func(data []byte) string {
		var ip dhcp.IPv4Address
		copy(ip[:], data[0:4])
		return ip.String()
	}

	mac6 := func(b []byte) string {
		var buf bytes.Buffer
		for i := range b {
			if i > 0 {
				buf.WriteString(":")
			}
			buf.WriteString(fmt.Sprintf("%02x", b[i]))
		}

		return buf.String()
	}

	opts := p.Options
	fmt.Println("Options:")
loop:
	for i := 0; i < len(opts); {
		o := opts[i]
		i++

		switch o {
		case dhcp.EndOption:
			fmt.Print("End Option")
			break loop
		case dhcp.PadOption:
			continue
		}

		length := int(opts[i])
		i++
		_, ok := options[o]
		if ok && options[o].Len >= 0 && options[o].Len != length {
			fmt.Printf("corrupted option (%d,%d)\n",
				options[o].Len, length)
			break loop
		}

		if name := options[o].Name; name != "" {
			fmt.Printf("%24s : ", options[o].Name)
		} else {
			fmt.Printf("%24d : ", o)
		}

		switch o {
		case dhcp.DHCPMessageType:
			fmt.Printf("%02x %s", opts[i], messageType[opts[i]])

		case dhcp.Router, dhcp.DomainNameServer, dhcp.NetBIOSNameServer:
			// Multiple IP addresses
			for n := 0; n < length; n += 4 {
				if n > 0 {
					fmt.Print(", ")
				}
				fmt.Print(ip4(opts[i+n : i+4+n]))
			}

		case dhcp.ServerIdentifier, dhcp.SubnetMask, dhcp.BroadcastAddress, dhcp.RequestedIPAddress:
			// Single IP address
			fmt.Print(ip4(opts[i:]))

		case dhcp.MaxDHCPMessageSize:
			// 16-bit integer
			fmt.Print(b16(opts[i:]))

		case dhcp.IPAddressLeaseTime, dhcp.RenewalTimeValue, dhcp.RebindingTimeValue:
			// 32-bit integer
			fmt.Print(b32(opts[i:]))

		case dhcp.HostName, dhcp.DomainName, dhcp.WebProxyServer:
			// String
			fmt.Printf("%q", string(opts[i:i+length]))

		case dhcp.DomainSearch:
			// Compressed domain names (RFC 1035)
			if s, err := dncomp.Decode(opts[i : i+length]); err != nil {
				fmt.Print(s)
			}

		case dhcp.ClientIdentifier:
			// Types according to RFC 1700
			switch opts[i] {
			case 1:
				fmt.Print(mac6(opts[i+1 : i+7]))
			default:
				fmt.Printf("type %d (len %d)", opts[i], length-1)
			}

		case dhcp.VendorSpecific, dhcp.VendorClassIdentifier:
			// Dump data
			fmt.Printf("%q", opts[i:i+length])

		case dhcp.ParameterRequestList:
			// Parameter list
			for i, p := range opts[i : i+length] {
				if i > 0 {
					fmt.Printf("\n%24s   ", "")
				}
				fmt.Printf("%02x %s", p, options[p].Name)
			}

		case dhcp.ClientFQDN:
			// Client FQDN format
			c := []byte{'-', '-', '-', '-'}
			d := []byte{'N', 'E', 'O', 'S'}
			for j := range c {
				if opts[j]&(1<<(3-uint(j))) != 0 {
					c[j] = d[j]
				}
			}
			fmt.Printf("%s %02x %02x ", string(c), opts[i+1],
				opts[i+2])
			if opts[i]&0x04 == 0 {
				fmt.Printf("%q", string(opts[i+3:]))
			} else {
				fmt.Printf("%q", wireString(opts[i+3:]))
			}
		}
		fmt.Println()

		i += length
	}
}

func showPacket(p *dhcp.Packet) {
	fmt.Printf("Transaction ID    : %#08x\n", p.Xid)
	fmt.Printf("Client IP address : %s\n", p.Ciaddr.String())
	fmt.Printf("Your IP address   : %s\n", p.Yiaddr.String())
	fmt.Printf("Server IP address : %s\n", p.Siaddr.String())
	fmt.Printf("Relay IP address  : %s\n", p.Giaddr.String())

	mac := p.Chaddr.MACAddress().String()
	fmt.Printf("Client MAC address: %s (%s)\n", mac, getVendor(mac))

	showOptions(p)

	fmt.Println()
}
