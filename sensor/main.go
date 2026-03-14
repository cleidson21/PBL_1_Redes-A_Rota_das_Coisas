package main

import (
	"fmt"
	"math/rand"
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

	// Define uma temperatura inicial realista
	temperaturaAtual := 25.0

	for {
		// Varia a temperatura suavemente entre -0.5 e +0.5 graus
		variacao := (rand.Float64() * 1.0) - 0.5
		temperaturaAtual += variacao

		// Cria limites para não congelar nem pegar fogo (ex: entre 18°C e 35°C)
		if temperaturaAtual < 18.0 {
			temperaturaAtual = 18.0
		} else if temperaturaAtual > 35.0 {
			temperaturaAtual = 35.0
		}

		mensagem := fmt.Sprintf("%.2f", temperaturaAtual)
		fmt.Printf("Enviando telemetria: %s°C\n", mensagem)

		// Envia o pacote UDP
		_, err := conn.Write([]byte(mensagem))
		if err != nil {
			fmt.Printf("Erro de rede: %v\n", err)
		}

		// Envio contínuo (a cada 500ms para simular alto volume de telemetria)
		time.Sleep(500 * time.Millisecond)
	}
}