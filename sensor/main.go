package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	addrEnv := os.Getenv("SERVER_ADDR")
	if addrEnv == "" {
		addrEnv = "localhost:8080"
	}

	servidorAddr, err := net.ResolveUDPAddr("udp", addrEnv)
	if err != nil {
		fmt.Printf("Erro ao resolver endereço: %v\n", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, servidorAddr)
	if err != nil {
		fmt.Printf("Erro ao conectar: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Printf("Sensor de Temperatura iniciado. Enviando para %s via UDP.\n", addrEnv)

	// Define uma temperatura inicial realista e a taxa de variação fixa
	temperaturaAtual := 25.0
	variacao := 0.33

	for {
		// Aplica a variação atual (sobe ou desce)
		temperaturaAtual += variacao

		if temperaturaAtual >= 40.0 {
			temperaturaAtual = 40.0
			variacao = -0.33
		} else if temperaturaAtual <= 16.0 {
			temperaturaAtual = 16.0
			variacao = 0.33
		}

		mensagem := fmt.Sprintf("%.2f", temperaturaAtual)
		fmt.Printf("Enviando temperatura: %s°C\n", mensagem)

		// Envia o pacote UDP
		_, err := conn.Write([]byte(mensagem))
		if err != nil {
			fmt.Printf("Erro de rede: %v\n", err)
		}

		// Envio contínuo (a cada 500ms)
		time.Sleep(500 * time.Millisecond)
	}
}
