package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	// Identidade logica do atuador enviada ao integrador.
	atuadorID := os.Getenv("ATUADOR_ID")
	if atuadorID == "" {
		atuadorID = "SALA_1"
	}

	tipoAtuador := os.Getenv("ATUADOR_TIPO")
	if tipoAtuador == "" {
		tipoAtuador = "LED"
	}

	// Endereco do integrador usado para registro e recebimento de comandos.
	integradorAddr := os.Getenv("INTEGRADOR_ADDR")
	if integradorAddr == "" {
		integradorAddr = "localhost:8082"
	}

	for {
		conn, err := net.Dial("tcp", integradorAddr)
		if err != nil {
			fmt.Printf("⚠️ Integrador offline. Tentando novamente em 5 segundos... (%v)\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		fmt.Printf("⚙️  [%s] %s Iniciado! Conectado em %s\n", atuadorID, tipoAtuador, integradorAddr)

		// Registra o atuador no gateway com o formato REG|TIPO|ID.
		fmt.Fprintf(conn, "REG|%s|%s\n", tipoAtuador, atuadorID)

		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			comando := strings.TrimSpace(scanner.Text())
			partes := strings.Split(comando, " ")
			acao := partes[0]

			switch acao {
			case "LIGAR":
				fmt.Printf("💡 [%s] Lâmpada ACESA...\n", atuadorID)
				// Resposta padrao enviada ao integrador para ser repassada ao cliente.
				fmt.Fprintf(conn, "ACK|%s|%s|LIGADO\n", tipoAtuador, atuadorID)
			case "DESLIGAR":
				fmt.Printf("🌑 [%s] Lâmpada APAGADA...\n", atuadorID)
				fmt.Fprintf(conn, "ACK|%s|%s|DESLIGADO\n", tipoAtuador, atuadorID)
			default:
				// Comandos fora do contrato sao apenas reportados no log.
				fmt.Printf("⚠️ Comando desconhecido para Lâmpada: %s\n", comando)
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("⚠️ Conexão com integrador interrompida: %v\n", err)
		}
		conn.Close()
		fmt.Println("❌ Conexão perdida. Iniciando reconexão...")
		time.Sleep(3 * time.Second)
	}
}
