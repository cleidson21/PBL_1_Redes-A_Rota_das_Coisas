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

// ESTRUTURA DE DADOS (O "Cérebro" do Sistema)

// EstadoSala guarda todas as informações de um ambiente específico
type EstadoSala struct {
	TemperaturaAtual float64
	TemperaturaAlvo  float64
	ArLigado         bool
	LampadaLigada    bool
	ModoAuto         bool
	UltimoEvento     string
}

// O Dicionário (Map) que guarda o estado de cada sala dinamicamente
var (
	mu    sync.RWMutex
	salas = make(map[string]*EstadoSala)
)

// getSalaSegura busca uma sala no mapa. Se não existir, cria uma com valores padrão.
// ATENÇÃO: Essa função já deve ser chamada com o Mutex travado!
func getSalaSegura(id string) *EstadoSala {
	if _, existe := salas[id]; !existe {
		salas[id] = &EstadoSala{
			TemperaturaAtual: 0.0,
			TemperaturaAlvo:  24.0,
			ArLigado:         false,
			LampadaLigada:    false,
			ModoAuto:         true,
			UltimoEvento:     "Nenhum",
		}
	}
	return salas[id]
}

// FUNÇÃO PRINCIPAL E INTERFACE COM O USUÁRIO (CLI)
func main() {
	addrEnv := os.Getenv("INTEGRADOR_ADDR")
	if addrEnv == "" {
		addrEnv = "localhost:8083" // Nova porta do Integrador para Clientes
	}

	conn, err := net.Dial("tcp", addrEnv)
	if err != nil {
		fmt.Printf("❌ Erro ao conectar ao Integrador em %s: %v\n", addrEnv, err)
		return
	}
	defer conn.Close()

	fmt.Println("✅ Conectado com sucesso ao Gateway Integrador!")

	// INICIA O CÉREBRO EM SEGUNDO PLANO
	// Ao invés de ler a rede aqui no loop principal, jogamos isso para uma Goroutine!
	go ouvirRedeEProcessarLogica(conn)

	// O LOOP DO MENU (Interação com o Usuário)
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n===================================")
		fmt.Println("❄️  PAINEL MULTI-SALA IoT ❄️")
		fmt.Println("===================================")
		fmt.Println("[1] Ver Status de Todas as Salas")
		fmt.Println("[2] Ligar/Desligar Ar (Manual)")
		fmt.Println("[3] Ligar/Desligar Modo Automático")
		fmt.Println("[4] Definir Nova Temperatura Alvo")
		fmt.Println("[5] Ligar/Desligar Lâmpada (Manual)")
		fmt.Println("[0] Sair")
		fmt.Println("===================================")
		fmt.Print("Escolha uma opção: ")

		opcao, _ := reader.ReadString('\n')
		opcao = strings.TrimSpace(opcao)

		switch opcao {
		case "1":
			imprimirPainel()
		case "2":
			fmt.Print("Digite o NOME DA SALA (ex: SALA_1): ")
			idSala, _ := reader.ReadString('\n')
			idSala = strings.TrimSpace(idSala)

			fmt.Print("Digite a AÇÃO (LIGAR ou DESLIGAR): ")
			acao, _ := reader.ReadString('\n')
			acao = strings.TrimSpace(strings.ToUpper(acao))

			mu.Lock()
			sala := getSalaSegura(idSala)
			sala.ModoAuto = false
			mu.Unlock()

			// ENVIAMOS COM O PREFIXO "AR_"
			fmt.Fprintf(conn, "AR_%s|%s\n", idSala, acao)
			fmt.Println("⏳ Comando enviado para o Ar-Condicionado! (Modo Auto desativado)")

		case "3":
			fmt.Print("Digite o NOME DA SALA (ex: SALA_1): ")
			idSala, _ := reader.ReadString('\n')
			idSala = strings.TrimSpace(idSala)

			mu.Lock()
			sala := getSalaSegura(idSala)
			sala.ModoAuto = !sala.ModoAuto // Inverte o valor atual
			statusAuto := sala.ModoAuto
			mu.Unlock()

			if statusAuto {
				fmt.Println("✅ Modo Automático ATIVADO para a sala", idSala)
			} else {
				fmt.Println("🛑 Modo Automático DESATIVADO para a sala", idSala)
			}

		case "4":
			fmt.Print("Digite o NOME DA SALA (ex: SALA_1): ")
			idSala, _ := reader.ReadString('\n')
			idSala = strings.TrimSpace(idSala)

			fmt.Print("Digite a nova TEMPERATURA ALVO (ex: 22.5): ")
			tempStr, _ := reader.ReadString('\n')
			tempVal, err := strconv.ParseFloat(strings.TrimSpace(tempStr), 64)

			if err == nil {
				mu.Lock()
				sala := getSalaSegura(idSala)
				sala.TemperaturaAlvo = tempVal
				mu.Unlock()
				fmt.Printf("🎯 Alvo da %s alterado para %.1f°C\n", idSala, tempVal)
			} else {
				fmt.Println("❌ Valor de temperatura inválido.")
			}

		case "5":
			fmt.Print("Digite o NOME DA SALA (ex: SALA_1): ")
			idSala, _ := reader.ReadString('\n')
			idSala = strings.TrimSpace(idSala)

			fmt.Print("Digite a AÇÃO (LIGAR ou DESLIGAR): ")
			acao, _ := reader.ReadString('\n')
			acao = strings.TrimSpace(strings.ToUpper(acao))

			// ENVIAMOS COM O PREFIXO "LAMP_"
			fmt.Fprintf(conn, "LAMP_%s|%s\n", idSala, acao)
			fmt.Println("💡 Comando enviado para a Lâmpada!")

		case "0":
			fmt.Println("Desconectando do sistema...")
			return
		default:
			fmt.Println("⚠️ Opção inválida.")
		}
	}
}

// MOTOR DE REGRAS EM SEGUNDO PLANO (Goroutine)
func ouvirRedeEProcessarLogica(conn net.Conn) {
	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		mensagem := strings.TrimSpace(scanner.Text())
		partes := strings.Split(mensagem, "|")

		if len(partes) < 3 {
			continue // Ignora mensagens mal formatadas
		}

		tipoMsg := partes[0]

		mu.Lock()

		// RECEBEU TELEMETRIA (Ex: TELEMETRIA|TEMP|SALA_1|25.5)
		if tipoMsg == "TELEMETRIA" && partes[1] == "TEMP" {
			idSala := partes[2]
			tempAtual, _ := strconv.ParseFloat(partes[3], 64)

			sala := getSalaSegura(idSala)
			sala.TemperaturaAtual = tempAtual

			// MÁGICA DA HISTERESE AQUI:
			avaliarModoAutomatico(idSala, sala, conn)
		}

		// RECEBEU EVENTO (Ex: EVENTO|NFC|CATRACA_ENTRADA|CRACHA_USER_4091)
		if tipoMsg == "EVENTO" {
			idSala := partes[2]
			evento := partes[3]

			sala := getSalaSegura(idSala)
			sala.UltimoEvento = evento
		}

		// 3. RECEBEU CONFIRMAÇÃO DO ATUADOR (Ex: ACK|AR_CONDICIONADO|SALA_1|LIGADO)
		if tipoMsg == "ACK" && len(partes) >= 4 {
			tipoAtuador := partes[1] // AR_CONDICIONADO ou LAMPADA
			idSala := partes[2]      // SALA_1
			acao := partes[3]        // LIGADO ou DESLIGADO

			sala := getSalaSegura(idSala)

			if tipoAtuador == "AR_CONDICIONADO" {
				sala.ArLigado = (acao == "LIGADO")
			} else if tipoAtuador == "LAMPADA" {
				sala.LampadaLigada = (acao == "LIGADO")
			}
		}

		mu.Unlock()
	}
}

// avaliarModoAutomatico executa a regra de negócio.
// ATENÇÃO: Essa função pressupõe que o Mutex já está travado por quem a chamou.
func avaliarModoAutomatico(id string, sala *EstadoSala, conn net.Conn) {
	if !sala.ModoAuto {
		return // Se estiver no manual, o cérebro não faz nada
	}

	limiteSuperior := sala.TemperaturaAlvo + 1.0
	limiteInferior := sala.TemperaturaAlvo - 1.0

	// Se esquentou demais e o ar está desligado -> Manda Ligar
	if sala.TemperaturaAtual >= limiteSuperior && !sala.ArLigado {
		fmt.Fprintf(conn, "%s|LIGAR\n", id)
	}

	// Se esfriou demais e o ar está ligado -> Manda Desligar
	if sala.TemperaturaAtual <= limiteInferior && sala.ArLigado {
		fmt.Fprintf(conn, "%s|DESLIGAR\n", id)
	}
}

// FUNÇÃO VISUAL: Imprime o estado de todas as salas na tela
func imprimirPainel() {
	mu.RLock()
	defer mu.RUnlock()

	fmt.Println("\n📊 === STATUS ATUAL DA REDE ===")
	if len(salas) == 0 {
		fmt.Println("Nenhum dado recebido ainda. Aguarde os sensores...")
		return
	}

	for id, sala := range salas {
		statusAr := "🔴 DESLIGADO"
		if sala.ArLigado {
			statusAr = "🟢 LIGADO"
		}

		modo := "✋ MANUAL"
		if sala.ModoAuto {
			modo = "🤖 AUTO"
		}

		fmt.Printf("📍 [%s] Temp: %.1f°C | Alvo: %.1f°C | Ar: %s | Modo: %s | Info Extra: %s\n",
			id, sala.TemperaturaAtual, sala.TemperaturaAlvo, statusAr, modo, sala.UltimoEvento)
	}
	fmt.Println("===============================")
}
