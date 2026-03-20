package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
)

var (
	// Mapa para guardar as conexões ativas dos Atuadores (Chave: ID do Atuador)
	muAtuadores sync.RWMutex
	atuadores   = make(map[string]net.Conn)

	// Mapa para guardar as conexões ativas dos Clientes/Dashboards
	muClientes sync.RWMutex
	clientes   = make(map[net.Conn]bool)
)

func main() {
	fmt.Println("🚀 Integrador Gateway (Broker) Iniciado!")
	fmt.Println("Ouvindo as seguintes portas:")
	fmt.Println("- UDP 8080: Sensores de Temperatura Contínuos")
	fmt.Println("- TCP 8081: Sensores de Eventos (NFC/Catraca)")
	fmt.Println("- TCP 8082: Atuadores (Ar Condicionado/Lâmpadas)")
	fmt.Println("- TCP 8083: Clientes (Dashboards e Controladores)")

	// Inicia as 4 portas em paralelo usando as Goroutines do Go
	go listenSensoresUDP()
	go listenSensoresTCP()
	go listenAtuadoresTCP()
	go listenClientesTCP()

	// Trava o programa principal infinitamente
	select {}
}

// PORTA 8080 (UDP) - Recebe Sensores de Temperatura Contínuos
func listenSensoresUDP() {
	addr, _ := net.ResolveUDPAddr("udp", ":8080")
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Printf("❌ Erro ao iniciar servidor UDP 8080: %v\n", err)
		return
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}

		mensagem := strings.TrimSpace(string(buffer[:n]))
		fmt.Printf("📥 [Sensor UDP] Recebeu: %s\n", mensagem)

		// Repassa a mensagem para todos os clientes conectados
		broadcastParaClientes("TLM|" + mensagem)
	}
}

// PORTA 8081 (TCP) - Recebe Sensores de Evento (NFC)
func listenSensoresTCP() {
	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		fmt.Printf("❌ Erro ao iniciar servidor TCP 8081 (Sensores): %v\n", err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		// Cada sensor TCP ganha uma rotina para manter o túnel aberto
		go func(c net.Conn) {
			defer c.Close()
			scanner := bufio.NewScanner(c)
			for scanner.Scan() {
				mensagem := strings.TrimSpace(scanner.Text())
				fmt.Printf("📥 [Sensor TCP] Recebeu: %s\n", mensagem)

				// OTIMIZAÇÃO: Usa o prefixo "EVT" (Evento)
				broadcastParaClientes("EVT|" + mensagem)
			}
		}(conn)
	}
}

// PORTA 8082 (TCP) - Recebe e Gerencia Atuadores
func listenAtuadoresTCP() {
	listener, err := net.Listen("tcp", ":8082")
	if err != nil {
		fmt.Printf("❌ Erro ao iniciar servidor TCP 8082 (Atuadores): %v\n", err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go manipularAtuador(conn)
	}
}

func manipularAtuador(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	var atuadorID string

	for scanner.Scan() {
		mensagem := strings.TrimSpace(scanner.Text())
		partes := strings.Split(mensagem, "|")

		// Se for uma mensagem de REGISTRO (Ex: REG|AC|SALA_1)
		if partes[0] == "REG" && len(partes) >= 3 {
			tipoAtuador := partes[1]
			idSala := partes[2]

			// Cria a chave única! Ex: AC_SALA_1 ou LED_SALA_1
			atuadorID = fmt.Sprintf("%s_%s", tipoAtuador, idSala)

			// Salva a conexão no Dicionário usando o Mutex para proteção
			muAtuadores.Lock()
			atuadores[atuadorID] = conn
			muAtuadores.Unlock()

			fmt.Printf("⚙️  [Atuador] %s registrado com sucesso!\n", atuadorID)
			continue
		}

		// Se for um Recibo/Confirmação (ACK) ou ERRO do Atuador, repassa para o Cliente
		if partes[0] == "ACK" || partes[0] == "ERRO" {
			fmt.Printf("📤 [Atuador -> Cliente] Repassando: %s\n", mensagem)
			broadcastParaClientes(mensagem)
		}
	}

	// Se o código chegar aqui, a conexão caiu. Removemos do Dicionário.
	if atuadorID != "" {
		muAtuadores.Lock()
		delete(atuadores, atuadorID)
		muAtuadores.Unlock()
		fmt.Printf("⚠️ [Atuador] %s desconectado.\n", atuadorID)
	}
	conn.Close()
}

// PORTA 8083 (TCP) - Recebe e Gerencia Clientes (Dashboards)
func listenClientesTCP() {
	listener, err := net.Listen("tcp", ":8083")
	if err != nil {
		fmt.Printf("❌ Erro ao iniciar servidor TCP 8083 (Clientes): %v\n", err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		// Registra o novo cliente
		muClientes.Lock()
		clientes[conn] = true
		muClientes.Unlock()

		fmt.Println("📱 [Cliente] Novo Dashboard conectado!")
		go manipularCliente(conn)
	}
}

func manipularCliente(conn net.Conn) {
	scanner := bufio.NewScanner(conn)

	// O Cliente enviará comandos no formato: ID_ATUADOR|COMANDO (Ex: AC_SALA_1|LIGAR)
	for scanner.Scan() {
		mensagem := strings.TrimSpace(scanner.Text())
		partes := strings.SplitN(mensagem, "|", 2) // Corta apenas no primeiro "|"

		if len(partes) == 2 {
			idDestino := partes[0]
			comando := partes[1]

			// Busca o túnel TCP daquele atuador específico na Lista Telefônica
			muAtuadores.RLock()
			atuadorConn, existe := atuadores[idDestino]
			muAtuadores.RUnlock()

			if existe {
				fmt.Printf("📤 [Cliente -> Atuador %s] Roteando comando: %s\n", idDestino, comando)
				fmt.Fprintf(atuadorConn, "%s\n", comando)
			} else {
				fmt.Printf("⚠️ [Cliente] Tentou enviar comando para %s, mas ele está offline.\n", idDestino)
				fmt.Fprintf(conn, "ERRO|GATEWAY|Atuador %s nao encontrado ou offline\n", idDestino)
			}
		}
	}

	// Se desconectar, remove o cliente da lista
	muClientes.Lock()
	delete(clientes, conn)
	muClientes.Unlock()
	fmt.Println("📱 [Cliente] Dashboard desconectado.")
	conn.Close()
}

// FUNÇÃO AUXILIAR: Envia mensagem para todos os painéis abertos
func broadcastParaClientes(mensagem string) {
	muClientes.RLock()
	defer muClientes.RUnlock()

	for conn := range clientes {
		fmt.Fprintf(conn, "%s\n", mensagem)
	}
}
