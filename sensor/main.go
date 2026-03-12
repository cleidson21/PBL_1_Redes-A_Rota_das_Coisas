package main

import (
	"fmt"
	"math/rand"
	"net"
	"time"
)

func main() {
	// Endereço do Integrador (localhost por enquanto, depois será o IP do container)
	servidorAddr, _ := net.ResolveUDPAddr("udp", "integrador:8080")
	conn, _ := net.DialUDP("udp", nil, servidorAddr)
	defer conn.Close()

	fmt.Println("Sensor iniciado... Enviando dados via UDP.")

	for {
		// Gera valor aleatório entre 20.0 e 40.0
		valor := 20.0 + rand.Float64()*(40.0-20.0)
		mensagem := fmt.Sprintf("%.2f", valor)

		fmt.Printf("Enviando temperatura: %s°C\n", mensagem)
		
		// Envia para o Integrador
		conn.Write([]byte(mensagem))

		// Aguarda 2 segundos para a próxima leitura
		time.Sleep(2 * time.Second)
	}
}
