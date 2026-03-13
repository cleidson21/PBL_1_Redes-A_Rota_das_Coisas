package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"
)

func main() {
	// Tenta ler o endereço do integrador da variável de ambiente SERVER_ADDR
	// Se não existir (vazio), usa "localhost:8080" como padrão para testes locais
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

	fmt.Printf("Sensor iniciado... Enviando dados para %s via UDP.\n", addrEnv)

	for {
		// Gera valor aleatório entre 20.0 e 40.0
		valor := 20.0 + rand.Float64()*(40.0-20.0)
		mensagem := fmt.Sprintf("%.2f", valor)

		fmt.Printf("Enviando temperatura: %s°C\n", mensagem)

		// Envia para o Integrador
		_, err := conn.Write([]byte(mensagem))
		if err != nil {
			fmt.Printf("Erro ao enviar dado: %v\n", err)
		}

		// Aguarda 1 segundo para a próxima leitura
		time.Sleep(1 * time.Second)
	}
}
