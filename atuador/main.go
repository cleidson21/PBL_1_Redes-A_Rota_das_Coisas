package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

func main() {
	// O Atuador será um servidor TCP aguardando comandos
	porta := ":8081"
	listener, err := net.Listen("tcp", porta)
	if err != nil {
		fmt.Printf("Erro ao iniciar atuador TCP: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Printf("Atuador de Refrigeração iniciado. Aguardando comandos na porta %s (TCP)...\n", porta)

	for {
		// Fica travado aqui até receber uma nova conexão do Integrador
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Erro ao aceitar conexão: %v\n", err)
			continue
		}

		// A palavra 'go' cria uma Goroutine! 
		// O sistema lida com esse cliente em paralelo e volta a ouvir a porta imediatamente.
		go manipularConexao(conn)
	}
}

// Função que processa os comandos recebidos
func manipularConexao(conn net.Conn) {
	defer conn.Close()
	clienteAddr := conn.RemoteAddr().String()
	fmt.Printf("\n[Nova Conexão] Sistema Integrador conectado: %s\n", clienteAddr)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		comando := strings.TrimSpace(scanner.Text())
		fmt.Printf("-> Comando Recebido: %s\n", comando)

		// Lógica simples do Ar-Condicionado (Separa a ação do valor)
		partes := strings.Split(comando, " ")
		acao := partes[0]

		switch acao {
		case "LIGAR":
			fmt.Println("❄️ AÇÃO: Ligando o compressor do ar-condicionado.")
			conn.Write([]byte("OK: Sistema Ligado\n")) // Confirmação de volta (ACK)
			
		case "DESLIGAR":
			fmt.Println("🛑 AÇÃO: Desligando o ar-condicionado.")
			conn.Write([]byte("OK: Sistema Desligado\n"))
			
		case "SET_TEMP":
			if len(partes) > 1 {
				fmt.Printf("🌡️ AÇÃO: Ajustando termostato para %s°C.\n", partes[1])
				conn.Write([]byte(fmt.Sprintf("OK: Temperatura ajustada para %s\n", partes[1])))
			} else {
				conn.Write([]byte("ERRO: Falta o valor da temperatura\n"))
			}
			
		default:
			fmt.Println("⚠️ Comando desconhecido ignorado.")
			conn.Write([]byte("ERRO: Comando invalido\n"))
		}
	}

	fmt.Printf("[Desconectado] Conexão com %s encerrada.\n", clienteAddr)
}