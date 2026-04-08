package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func habilitarKeepAlive(conn net.Conn) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return
	}
	_ = tcpConn.SetKeepAlive(true)
	_ = tcpConn.SetKeepAlivePeriod(30 * time.Second)
}

func main() {
	// Identidade logica do atuador enviada ao integrador.
	atuadorID := os.Getenv("ATUADOR_ID")
	if atuadorID == "" {
		atuadorID = "SALA_1"
	}

	tipoAtuador := os.Getenv("ATUADOR_TIPO")
	if tipoAtuador == "" {
		tipoAtuador = "AC"
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
		habilitarKeepAlive(conn)

		fmt.Printf("⚙️  [%s] %s Iniciado! Conectado em %s\n", atuadorID, tipoAtuador, integradorAddr)

		// Registra o atuador no gateway com o formato REG|TIPO|ID.
		fmt.Fprintf(conn, "REG|%s|%s\n", tipoAtuador, atuadorID)

		estadoAtual := "DESLIGADO"

		// Canal para avisar a Goroutine de Heartbeat que deve morrer.
		done := make(chan bool)

		// Goroutine de Heartbeat: envia estado a cada 10 segundos.
		go func() {
			for {
				select {
				case <-time.After(10 * time.Second):
					fmt.Fprintf(conn, "ACK|%s|%s|%s\n", tipoAtuador, atuadorID, estadoAtual)
				case <-done:
					log.Printf("🛑 Parando Heartbeat antigo do %s...\n", atuadorID)
					return
				}
			}
		}()

		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			comando := strings.TrimSpace(scanner.Text())
			partes := strings.Split(comando, " ")
			acao := partes[0]

			switch acao {
			case "LIGAR":
				fmt.Printf("❄️ [%s] LIGANDO compressor do Ar...\n", atuadorID)
				estadoAtual = "LIGADO"
				fmt.Fprintf(conn, "ACK|%s|%s|LIGADO\n", tipoAtuador, atuadorID)
			case "DESLIGAR":
				fmt.Printf("🛑 [%s] DESLIGANDO Ar-Condicionado...\n", atuadorID)
				estadoAtual = "DESLIGADO"
				fmt.Fprintf(conn, "ACK|%s|%s|DESLIGADO\n", tipoAtuador, atuadorID)
			case "SET_TEMP":
				if len(partes) > 1 {
					fmt.Printf("🌡️ [%s] Ajustando termostato para %s°C\n", atuadorID, partes[1])
					fmt.Fprintf(conn, "ACK|%s|%s|TEMP_SETADA_%s\n", tipoAtuador, atuadorID, partes[1])
				}
			default:
				log.Printf("Comando desconhecido para AC: %s\n", comando)
			}
		}

		// SE CHEGOU AQUI, A CONEXÃO CAIU! Matamos a Goroutine de Heartbeat.
		done <- true

		if err := scanner.Err(); err != nil {
			fmt.Printf("⚠️ Conexão com integrador interrompida: %v\n", err)
		}
		conn.Close()
		fmt.Println("❌ Conexão perdida. Iniciando reconexão...")
		time.Sleep(3 * time.Second)
	}
}
