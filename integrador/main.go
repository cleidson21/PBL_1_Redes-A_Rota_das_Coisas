package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

// EstadoSistema guarda todas as variáveis importantes.
// O Mutex (mu) impede que duas conexões tentem alterar os dados no exato mesmo milissegundo.
type EstadoSistema struct {
	mu               sync.Mutex
	temperaturaAtual float64
	arLigado         bool
	modoAuto         bool
	tempAlvo         float64
}

// Inicializa o sistema no modo automático buscando 24°C
var estado = EstadoSistema{
	modoAuto: true,
	tempAlvo: 24.0,
}

var atuadorAddr string

func main() {
	atuadorAddr = os.Getenv("ATUADOR_ADDR")
	if atuadorAddr == "" {
		atuadorAddr = "localhost:8081"
	}

	fmt.Println("🧠 Sistema Integrador Iniciado!")

	// A magia do Go: Iniciamos dois servidores rodando em paralelo!
	go iniciarServidorUDP()        // Fica ouvindo o Sensor o tempo todo
	go iniciarServidorClienteTCP() // Fica aguardando conexões do Dashboard

	// Mantém o programa principal rodando infinitamente
	select {}
}

// ---------------------------------------------------------
// 1. SERVIDOR UDP (Ouve o Sensor)
// ---------------------------------------------------------
func iniciarServidorUDP() {
	addr, _ := net.ResolveUDPAddr("udp", ":8080")
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Printf("Erro UDP: %v\n", err)
		return
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}

		tempStr := strings.TrimSpace(string(buffer[:n]))
		tempFloat, err := strconv.ParseFloat(tempStr, 64)
		if err == nil {
			// Trava o Mutex rápido só para atualizar a temperatura lida
			estado.mu.Lock()
			estado.temperaturaAtual = tempFloat
			estado.mu.Unlock()

			// Chama a lógica de avaliação
			avaliarModoAutomatico()
		}
	}
}

// ---------------------------------------------------------
// 2. LÓGICA DE CONTROLE (O Cérebro)
// ---------------------------------------------------------
func avaliarModoAutomatico() {
	estado.mu.Lock()
	defer estado.mu.Unlock()

	// Só atua sozinho se estiver no modo Automático
	if !estado.modoAuto {
		return
	}

	// Lógica de resfriamento: Liga se passar 1 grau do alvo
	if estado.temperaturaAtual >= (estado.tempAlvo+1.0) && !estado.arLigado {
		fmt.Println("⚠️ AUTO: Temperatura subiu. Ligando ar-condicionado.")
		enviarComandoAtuador("LIGAR")
		enviarComandoAtuador(fmt.Sprintf("SET_TEMP %.1f", estado.tempAlvo))
		estado.arLigado = true
	}

	// Lógica de economia: Desliga se chegar 1 grau abaixo do alvo
	if estado.temperaturaAtual <= (estado.tempAlvo-1.0) && estado.arLigado {
		fmt.Println("✅ AUTO: Ambiente resfriado. Desligando compressor.")
		enviarComandoAtuador("DESLIGAR")
		estado.arLigado = false
	}
}

// ---------------------------------------------------------
// 3. SERVIDOR TCP (Ouve o Dashboard do Cliente)
// ---------------------------------------------------------
func iniciarServidorClienteTCP() {
	// Nova porta exclusiva para a interface do cliente
	listener, err := net.Listen("tcp", ":8082")
	if err != nil {
		fmt.Printf("Erro TCP Cliente: %v\n", err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		// Cada cliente novo ganha sua própria Goroutine
		go manipularCliente(conn)
	}
}

func manipularCliente(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		comando := strings.TrimSpace(scanner.Text())
		partes := strings.Split(comando, " ")
		acao := partes[0]

		estado.mu.Lock() // Trava o estado para o cliente mexer com segurança

		switch acao {
		case "STATUS":
			statusAr := "DESLIGADO"
			if estado.arLigado {
				statusAr = "LIGADO"
			}
			modo := "MANUAL"
			if estado.modoAuto {
				modo = "AUTO"
			}
			resposta := fmt.Sprintf("TEMP: %.2f | AR: %s | MODO: %s | ALVO: %.1f\n",
				estado.temperaturaAtual, statusAr, modo, estado.tempAlvo)
			conn.Write([]byte(resposta))

		case "AUTO":
			estado.modoAuto = true
			conn.Write([]byte("OK: Modo Automatico Ativado\n"))
			fmt.Println("📱 CLIENTE: Alterou para modo AUTOMÁTICO.")

		case "MANUAL":
			estado.modoAuto = false
			conn.Write([]byte("OK: Modo Manual Ativado\n"))
			fmt.Println("📱 CLIENTE: Alterou para modo MANUAL.")

		case "LIGAR":
			estado.modoAuto = false // Ligar na mão desativa o automático
			if !estado.arLigado {
				enviarComandoAtuador("LIGAR")
				estado.arLigado = true
			}
			conn.Write([]byte("OK: Ar Ligado (Modo Manual)\n"))
			fmt.Println("📱 CLIENTE: Ligou o ar manualmente.")

		case "DESLIGAR":
			estado.modoAuto = false
			if estado.arLigado {
				enviarComandoAtuador("DESLIGAR")
				estado.arLigado = false
			}
			conn.Write([]byte("OK: Ar Desligado (Modo Manual)\n"))
			fmt.Println("📱 CLIENTE: Desligou o ar manualmente.")

		case "SET_ALVO":
			if len(partes) > 1 {
				novoAlvo, err := strconv.ParseFloat(partes[1], 64)
				if err == nil {
					estado.tempAlvo = novoAlvo
					estado.modoAuto = true // Mudar o alvo reativa o automático
					enviarComandoAtuador(fmt.Sprintf("SET_TEMP %.1f", novoAlvo))
					conn.Write([]byte(fmt.Sprintf("OK: Novo alvo definido para %.1f\n", novoAlvo)))
					fmt.Printf("📱 CLIENTE: Novo alvo térmico -> %.1f\n", novoAlvo)
				}
			}
		}

		estado.mu.Unlock() // Libera o estado para o Sensor continuar atualizando

		// Força a avaliação logo após o cliente mudar alguma coisa
		if estado.modoAuto {
			avaliarModoAutomatico()
		}
	}
}

// ---------------------------------------------------------
// 4. FUNÇÃO AUXILIAR (Fala com o Atuador)
// ---------------------------------------------------------
func enviarComandoAtuador(comando string) {
	connTCP, err := net.Dial("tcp", atuadorAddr)
	if err != nil {
		fmt.Printf("❌ Falha de rede com Atuador: %v\n", err)
		return
	}
	defer connTCP.Close()
	fmt.Fprintf(connTCP, "%s\n", comando)
}
