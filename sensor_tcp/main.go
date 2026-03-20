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

	conn, err := net.Dial("tcp", addrEnv)
	if err != nil {
		fmt.Printf("❌ Erro ao conectar no Integrador TCP: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Printf("🪪 Sensor [%s] tipo [%s] iniciado! Enviando leituras para %s via TCP.\n", sensorID, sensorTipo, addrEnv)

	// Simula leituras de crachás
	crachas := []string{"USER_4091", "USER_1192", "USER_5583", "USER_9944"}

	for {
		// Sorteia um crachá e cria a mensagem no formato "SENSOR_TIPO
		crachaLido := crachas[rand.Intn(len(crachas))]

		mensagem := fmt.Sprintf("%s|%s|%s", sensorTipo, sensorID, crachaLido)

		fmt.Printf("Enviando leitura -> %s\n", mensagem)

		// Envia a mensagem para o Integrador TCP (a conexão já está estabelecida)
		fmt.Fprintf(conn, "%s\n", mensagem)

		// Espera um tempo aleatório entre 5 e 15 segundos para simular a próxima pessoa passando pela catraca
		tempoEspera := time.Duration(rand.Intn(10)+5) * time.Second
		time.Sleep(tempoEspera)
	}
}
