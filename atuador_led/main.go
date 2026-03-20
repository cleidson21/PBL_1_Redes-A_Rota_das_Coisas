package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	// Identidade do Atuador
	atuadorID := os.Getenv("ATUADOR_ID")
	if atuadorID == "" {
		atuadorID = "SALA_1"
	}

	tipoAtuador := os.Getenv("ATUADOR_TIPO")
	if tipoAtuador == "" {
		tipoAtuador = "LED"
	}

	// Configurações de Rede do Integrador
	integradorAddr := os.Getenv("INTEGRADOR_ADDR")
	if integradorAddr == "" {
		integradorAddr = "localhost:8082"
	}

	conn, err := net.Dial("tcp", integradorAddr)
	if err != nil {
		fmt.Printf("❌ Erro ao conectar no Integrador: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Printf("⚙️  [%s] %s Iniciado! Conectado em %s\n", atuadorID, tipoAtuador, integradorAddr)

	// Fica: REG|LED|SALA_1
	fmt.Fprintf(conn, "REG|%s|%s\n", tipoAtuador, atuadorID)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		comando := strings.TrimSpace(scanner.Text())
		partes := strings.Split(comando, " ")
		acao := partes[0]

		switch acao {
		case "LIGAR":
			fmt.Printf("💡 [%s] Lâmpada ACESA...\n", atuadorID)
			// Exemplo de saída: ACK|LED|SALA_1|LIGADO
			fmt.Fprintf(conn, "ACK|%s|%s|LIGADO\n", tipoAtuador, atuadorID)
		case "DESLIGAR":
			fmt.Printf("🌑 [%s] Lâmpada APAGADA...\n", atuadorID)
			fmt.Fprintf(conn, "ACK|%s|%s|DESLIGADO\n", tipoAtuador, atuadorID)
		default:
			fmt.Printf("⚠️ Comando desconhecido para Lâmpada: %s\n", comando)
		}
	}
}
