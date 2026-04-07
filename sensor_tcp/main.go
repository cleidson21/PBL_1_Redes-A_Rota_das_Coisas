package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"
)

func main() {
	// Endereco do integrador TCP. Usa um padrao local quando a variavel nao vem do ambiente.
	addrEnv := os.Getenv("SERVER_ADDR")
	if addrEnv == "" {
		addrEnv = "localhost:8081"
	}

	// Identificacao do sensor enviada junto com cada leitura.
	sensorID := os.Getenv("SENSOR_ID")
	if sensorID == "" {
		sensorID = "CATRACA_ENTRADA"
	}

	// Tipo da leitura produzida por este sensor.
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

	// Lista de identificadores usados para simular passagens na catraca.
	crachas := []string{"USER_4091", "USER_1192", "USER_5583", "USER_9944"}

	for {
		// Sorteia uma leitura e monta o payload esperado pelo integrador.
		crachaLido := crachas[rand.Intn(len(crachas))]

		mensagem := fmt.Sprintf("%s|%s|%s", sensorTipo, sensorID, crachaLido)

		fmt.Printf("Enviando leitura -> %s\n", mensagem)

		// TCP reutiliza a conexao aberta, entao aqui enviamos apenas a linha atual.
		fmt.Fprintf(conn, "%s\n", mensagem)

		// Intervalo aleatorio entre leituras para simular fluxo irregular de pessoas.
		tempoEspera := time.Duration(rand.Intn(10)+5) * time.Second
		time.Sleep(tempoEspera)
	}
}
