package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

func main() {
	// Endereço do atuador (lido da variável de ambiente ou usa localhost por padrão)
	atuadorAddr := os.Getenv("ATUADOR_ADDR")
	if atuadorAddr == "" {
		atuadorAddr = "localhost:8081"
	}

	// 1. Configura o Servidor UDP para ouvir a telemetria do Sensor
	addr, err := net.ResolveUDPAddr("udp", ":8080")
	if err != nil {
		fmt.Printf("Erro ao resolver endereço UDP: %v\n", err)
		return
	}

	connUDP, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Printf("Erro ao iniciar servidor UDP: %v\n", err)
		return
	}
	defer connUDP.Close()

	fmt.Println("Integrador iniciado. Ouvindo telemetria na porta 8080 (UDP)...")

	buffer := make([]byte, 1024)
	
	// Estado do ar-condicionado (para evitar mandar comandos repetidos)
	arCondicionadoLigado := false 

	for {
		n, remoteAddr, err := connUDP.ReadFromUDP(buffer)
		if err != nil {
			fmt.Printf("Erro ao ler do UDP: %v\n", err)
			continue
		}

		dadoStr := strings.TrimSpace(string(buffer[:n]))
		fmt.Printf("Recebido de %s: %s°C\n", remoteAddr, dadoStr)

		// Converte a string da temperatura para um número decimal (float64)
		temperatura, err := strconv.ParseFloat(dadoStr, 64)
		if err != nil {
			fmt.Printf("Erro ao converter temperatura: %v\n", err)
			continue
		}

		// --- LÓGICA DE CONTROLE AUTOMÁTICO (O Cérebro) ---
		
		// Se passar de 26°C e estiver desligado -> LIGA
		if temperatura >= 26.0 && !arCondicionadoLigado {
			fmt.Println("⚠️ ALERTA: Temperatura alta! Acionando ar-condicionado via TCP.")
			enviarComandoTCP(atuadorAddr, "LIGAR")
			enviarComandoTCP(atuadorAddr, "SET_TEMP 22")
			arCondicionadoLigado = true
		}

		// Se baixar de 22°C e estiver ligado -> DESLIGA
		if temperatura <= 22.0 && arCondicionadoLigado {
			fmt.Println("✅ AVISO: Temperatura ideal atingida! Desligando ar-condicionado via TCP.")
			enviarComandoTCP(atuadorAddr, "DESLIGAR")
			arCondicionadoLigado = false
		}
	}
}

// Função que abre a conexão TCP, envia o comando, lê a resposta e fecha
func enviarComandoTCP(endereco string, comando string) {
	connTCP, err := net.Dial("tcp", endereco)
	if err != nil {
		fmt.Printf("❌ Erro ao conectar com o Atuador (%s): %v\n", endereco, err)
		return
	}
	defer connTCP.Close()

	// Envia o comando com uma quebra de linha no final
	fmt.Fprintf(connTCP, "%s\n", comando)

	// Lê a confirmação do Atuador (opcional, mas prova que o TCP funcionou)
	resposta := make([]byte, 1024)
	n, _ := connTCP.Read(resposta)
	fmt.Printf("🔙 Resposta do Atuador: %s\n", strings.TrimSpace(string(resposta[:n])))
}