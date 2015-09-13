package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
)

var (
	stats Statistics
	cmd   map[string]func()
)

type Statistics struct {
	pkrec, pkproc uint
	pksent        uint
	count         map[string]uint // map mac to packet count
	msg           map[string]uint // map msg type to count
	vdc           map[string]uint // map vendor class to count
}

func init() {
	stats = Statistics{}
	stats.count = map[string]uint{}
	stats.msg = map[string]uint{}
	stats.vdc = map[string]uint{}

	cmd = map[string]func(){
		"discover": cmdDiscover,
		"snoop":    cmdSnoop,
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}

func setupSummary() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		summary()
		os.Exit(1)
	}()
}

func summary() {
	fmt.Println("\nPacket summary")
	fmt.Println("  Packets sent      :", stats.pksent)
	fmt.Println("  Packets received  :", stats.pkrec)
	fmt.Println("  Packets processed :", stats.pkproc)

	fmt.Println("\nMessage Types")
	for key, val := range stats.msg {
		fmt.Printf("  %-12.12s : %d\n", key, val)
	}

	fmt.Println("\nVendors")

	vcount := map[string]int{}
	for key, _ := range stats.count {
		v := VendorFromMAC(key)
		vcount[v]++
	}

	for key, val := range vcount {
		fmt.Printf("  %-8.8s : %d\n", key, val)
	}

	if len(stats.vdc) > 0 {
		fmt.Println("\nVendor classes")
		for key, val := range stats.vdc {
			fmt.Printf("  %-20.20s : %d\n", key, val)
		}
	}
}

func usage(c string) {

	cc := c
	if c == "" {
		cc = "<command>"
	}

	fmt.Fprintf(os.Stderr, "usage: %s %s [options]\n", os.Args[0], cc)

	if c == "" {
		fmt.Fprintf(os.Stderr, "available commands: ")
		var keys []string
		for key, _ := range cmd {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		fmt.Fprintf(os.Stderr, "%s\n", strings.Join(keys, " "))
	}

	flag.PrintDefaults()
}

func main() {
	if len(os.Args) < 2 {
		usage("")
		os.Exit(1)
	}

	if handle := cmd[os.Args[1]]; handle != nil {
		// remove command from argument list
		if len(os.Args) > 2 {
			os.Args = append(os.Args[:1], os.Args[2:]...)
		}
		handle()
		summary()
		os.Exit(0)
	}

	fmt.Fprintf(os.Stderr, "%s: %s: invalid command\n", os.Args[0], os.Args[1])
	os.Exit(1)

}
