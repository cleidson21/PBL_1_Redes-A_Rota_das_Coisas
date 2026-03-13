package main

import (
	"fmt"
	"net"
)

func main() {
	// ":8080" garante que o integrador ouça em todas as interfaces de rede do container
	addr, err := net.ResolveUDPAddr("udp", ":8080")
	if err != nil {
		fmt.Printf("Erro ao resolver endereço: %v\n", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Printf("Erro ao iniciar servidor UDP: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("Integrador iniciado. Ouvindo na porta 8080 (UDP)...")

	buffer := make([]byte, 1024)

	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Printf("Erro ao ler do UDP: %v\n", err)
			continue
		}

		fmt.Printf("Recebido de %s: %s°C\n", remoteAddr, string(buffer[:n]))
	}
}
