package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	// Configurações de Rede
	addrEnv := os.Getenv("SERVER_ADDR")
	if addrEnv == "" {
		addrEnv = "localhost:8080"
	}

	// Identidade do Sensor (Ambiente e Tipo)
	// Pega o nome do sensor (Ex: SALA_1, SALA_2)
	sensorID := os.Getenv("SENSOR_ID")
	if sensorID == "" {
		sensorID = "SALA_1"
	}

	// Pega o tipo de grandeza física (Ex: T para Temperatura, U para Umidade)
	sensorTipo := os.Getenv("SENSOR_TIPO")
	if sensorTipo == "" {
		sensorTipo = "T"
	}

	// Conexão UDP
	servidorAddr, err := net.ResolveUDPAddr("udp", addrEnv)
	if err != nil {
		fmt.Printf("❌ Erro ao resolver endereço: %v\n", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, servidorAddr)
	if err != nil {
		fmt.Printf("❌ Erro ao conectar: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Printf("📡 Sensor [%s] tipo [%s] iniciado! Enviando telemetria para %s via UDP.\n", sensorID, sensorTipo, addrEnv)

	// Lógica de Simulação de Dados
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

		// Formato: TIPO|ID|VALOR -> Exemplo na rede: T|SALA_1|25.50
		mensagem := fmt.Sprintf("%s|%s|%.2f", sensorTipo, sensorID, temperaturaAtual)
		fmt.Printf("Enviando -> %s\n", mensagem)

		// Envia o pacote UDP (Fire and Forget)
		_, err := conn.Write([]byte(mensagem))
		if err != nil {
			fmt.Printf("⚠️ Erro de rede: %v\n", err)
		}

		// Envio contínuo (a cada 500ms)
		time.Sleep(500 * time.Millisecond)
	}
}
