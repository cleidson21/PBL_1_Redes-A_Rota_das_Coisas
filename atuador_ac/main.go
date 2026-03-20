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
		atuadorID = "ATUADOR_PADRAO_1"
	}

	// Pode ser AR_CONDICIONADO ou LAMPADA
	tipoAtuador := os.Getenv("ATUADOR_TIPO")
	if tipoAtuador == "" {
		tipoAtuador = "AR_CONDICIONADO"
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

	// Manda a mensagem de Registro para o Integrador
	fmt.Fprintf(conn, "REGISTRO|AR_CONDICIONADO|%s\n", atuadorID)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		comando := strings.TrimSpace(scanner.Text())
		partes := strings.Split(comando, " ")
		acao := partes[0]

		switch acao {
		case "LIGAR":
			fmt.Printf("❄️ [%s] LIGANDO compressor do Ar...\n", atuadorID)
			fmt.Fprintf(conn, "ACK|AR_CONDICIONADO|%s|LIGADO\n", atuadorID)
		case "DESLIGAR":
			fmt.Printf("🛑 [%s] DESLIGANDO Ar-Condicionado...\n", atuadorID)
			fmt.Fprintf(conn, "ACK|AR_CONDICIONADO|%s|DESLIGADO\n", atuadorID)
		case "SET_TEMP":
			if len(partes) > 1 {
				fmt.Printf("🌡️ [%s] Ajustando termostato para %s°C\n", atuadorID, partes[1])
				fmt.Fprintf(conn, "ACK|AR_CONDICIONADO|%s|TEMP_SETADA_%s\n", atuadorID, partes[1])
			}
		default:
			fmt.Printf("⚠️ Comando desconhecido para AC: %s\n", comando)
		}
	}
}
