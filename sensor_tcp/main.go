package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"
)

func main() {
	// Agora os Sensores TCP vão mandar dados para a porta 8081
	addrEnv := os.Getenv("SERVER_ADDR")
	if addrEnv == "" {
		addrEnv = "localhost:8081"
	}

	sensorID := os.Getenv("SENSOR_ID")
	if sensorID == "" {
		sensorID = "CATRACA_ENTRADA"
	}

	sensorTipo := os.Getenv("SENSOR_TIPO")
	if sensorTipo == "" {
		sensorTipo = "NFC"
	}

	// Conexão TCP (diferente do UDP, aqui criamos o túnel contínuo)
	conn, err := net.Dial("tcp", addrEnv)
	if err != nil {
		fmt.Printf("❌ Erro ao conectar no Integrador TCP: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Printf("🪪 %s [%s] iniciado! Enviando leituras para %s via TCP.\n", sensorTipo, sensorID, addrEnv)

	// Simula leituras de crachás aleatórios
	crachas := []string{"USER_4091", "USER_1192", "USER_5583", "USER_9944"}

	for {
		// Sorteia um crachá e cria a mensagem
		crachaLido := crachas[rand.Intn(len(crachas))]
		mensagem := fmt.Sprintf("%s|%s|CRACHA_%s\n", sensorTipo, sensorID, crachaLido)

		fmt.Printf("Enviando leitura -> %s", mensagem)

		// Envia pelo túnel TCP
		fmt.Fprintf(conn, "%s\n", mensagem)

		// Espera um tempo aleatório entre 5 e 15 segundos para simular a próxima pessoa
		tempoEspera := time.Duration(rand.Intn(10)+5) * time.Second
		time.Sleep(tempoEspera)
	}
}
