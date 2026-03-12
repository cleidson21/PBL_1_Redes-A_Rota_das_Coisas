package main

import (
	"fmt"
	"net"
)

func main() {
	addr, _ := net.ResolveUDPAddr("udp", ":8080")
	conn, _ := net.ListenUDP("udp", addr)
	defer conn.Close()

	fmt.Println("Integrador ouvindo na porta 8080 (UDP)...")

	buffer := make([]byte, 1024)

	for {
		n, remoteAddr, _ := conn.ReadFromUDP(buffer)
		fmt.Printf("Recebido de %s: %s°C\n", remoteAddr, string(buffer[:n]))
	}
}
