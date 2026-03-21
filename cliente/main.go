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
type EstadoSala struct {
	TemSensorTemp    bool // Descobre se a sala tem termômetro
	TemperaturaAtual float64
	TemperaturaAlvo  float64

	TemAC    bool // Descobre se a sala tem Ar-Condicionado
	ArLigado bool
	ModoAuto bool

	TemLampada    bool // Descobre se a sala tem Lâmpada
	LampadaLigada bool

	TemCatraca   bool // Descobre se é um ambiente com Catraca/NFC
	UltimoEvento string
}

// O Dicionário (Map) que guarda o estado de cada sala dinamicamente
var (
	mu    sync.RWMutex
	salas = make(map[string]*EstadoSala)
)

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

// FUNÇÃO PRINCIPAL E INTERFACE COM O UTILIZADOR (CLI)
func main() {
	addrEnv := os.Getenv("INTEGRADOR_ADDR")
	if addrEnv == "" {
		addrEnv = "localhost:8083" // Porta do Integrador para Clientes
	}

	conn, err := net.Dial("tcp", addrEnv)
	if err != nil {
		fmt.Printf("❌ Erro ao ligar ao Integrador em %s: %v\n", addrEnv, err)
		return
	}
	defer conn.Close()

	fmt.Println("✅ Ligado com sucesso ao Gateway Integrador!")

	// INICIA O CÉREBRO EM SEGUNDO PLANO
	go ouvirRedeEProcessarLogica(conn)

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

			fmt.Fprintf(conn, "AC_%s|%s\n", idSala, acao)
			fmt.Println("⏳ Comando enviado para o Ar-Condicionado! (Modo Auto desativado)")

		case "3":
			fmt.Print("Digite o NOME DA SALA (ex: SALA_1): ")
			idSala, _ := reader.ReadString('\n')
			idSala = strings.TrimSpace(idSala)

			mu.Lock()
			sala := getSalaSegura(idSala)
			sala.ModoAuto = !sala.ModoAuto
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

				fmt.Fprintf(conn, "AC_%s|SET_TEMP %.1f\n", idSala, tempVal)
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

			fmt.Fprintf(conn, "LED_%s|%s\n", idSala, acao)
			fmt.Println("💡 Comando enviado para a Lâmpada!")

		case "0":
			fmt.Println("A desligar do sistema...")
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
			continue
		}

		tipoMsg := partes[0]

		mu.Lock()

		// RECEBEU TELEMETRIA (Ex: TLM|T|SALA_1|25.5)
		if tipoMsg == "TLM" && partes[1] == "T" {
			idSala := partes[2]
			tempAtual, _ := strconv.ParseFloat(partes[3], 64)

			sala := getSalaSegura(idSala)
			sala.TemSensorTemp = true
			sala.TemperaturaAtual = tempAtual

			avaliarModoAutomatico(idSala, sala, conn)
		}

		// RECEBEU EVENTO (Ex: EVT|NFC|CATRACA_ENTRADA|USER_4091)
		if tipoMsg == "EVT" {
			idSala := partes[2]
			evento := partes[3]

			sala := getSalaSegura(idSala)
			sala.TemCatraca = true
			sala.UltimoEvento = evento
		}

		// RECEBEU CONFIRMAÇÃO DO ATUADOR (Ex: ACK|AC|SALA_1|LIGADO ou ACK|LED|SALA_1|LIGADO)
		if tipoMsg == "ACK" && len(partes) >= 4 {
			tipoAtuador := partes[1]
			idSala := partes[2]
			acao := partes[3]

			sala := getSalaSegura(idSala)

			if tipoAtuador == "AC" {
				sala.TemAC = true
				sala.ArLigado = (acao == "LIGADO")
			} else if tipoAtuador == "LED" {
				sala.TemLampada = true
				sala.LampadaLigada = (acao == "LIGADO")
			}
		}

		if tipoMsg == "ERRO" && len(partes) >= 3 {
			origem := partes[1]  // De onde veio o erro (GATEWAY, AC ou LED)
			detalhe := partes[2] // O texto do erro

			// Dá um aviso visual no terminal
			fmt.Printf("\n❌ [FALHA DE COMANDO - %s] %s\n", origem, detalhe)

			// Reimprimimos a linha do menu para não quebrar o visual da tela
			fmt.Print("Escolha uma opção: ")
		}

		mu.Unlock()
	}
}

func avaliarModoAutomatico(id string, sala *EstadoSala, conn net.Conn) {
	if !sala.ModoAuto {
		return
	}

	limiteSuperior := sala.TemperaturaAlvo + 1.0
	limiteInferior := sala.TemperaturaAlvo - 1.0

	if sala.TemperaturaAtual >= limiteSuperior && !sala.ArLigado {
		fmt.Fprintf(conn, "AC_%s|LIGAR\n", id)
	}

	if sala.TemperaturaAtual <= limiteInferior && sala.ArLigado {
		fmt.Fprintf(conn, "AC_%s|DESLIGAR\n", id)
	}
}

// FUNÇÃO VISUAL: Imprime o estado de forma dinâmica (Só mostra o que existe!)
func imprimirPainel() {
	mu.RLock()
	defer mu.RUnlock()

	fmt.Println("\n📊 === STATUS ATUAL DA REDE ===")
	if len(salas) == 0 {
		fmt.Println("Nenhum dado recebido ainda. Aguarde pelos sensores...")
		return
	}

	for id, sala := range salas {
		var blocos []string // Lista para juntar as informações ativas

		// Só mostra Temperatura e Alvo se existir um sensor a enviar dados
		if sala.TemSensorTemp {
			blocos = append(blocos, fmt.Sprintf("🌡️ Temp: %.1f°C", sala.TemperaturaAtual))
			blocos = append(blocos, fmt.Sprintf("🎯 Alvo: %.1f°C", sala.TemperaturaAlvo))

			// Só faz sentido mostrar Modo Automático se a sala tiver Temperatura
			modo := "✋ MANUAL"
			if sala.ModoAuto {
				modo = "🤖 AUTO"
			}
			blocos = append(blocos, fmt.Sprintf("⚙️ Modo: %s", modo))
		}

		// Só mostra Ar-Condicionado se um responder
		if sala.TemAC {
			statusAr := "🔴 DESLIGADO"
			if sala.ArLigado {
				statusAr = "🟢 LIGADO"
			}
			blocos = append(blocos, fmt.Sprintf("❄️ Ar: %s", statusAr))
		}

		// Só mostra Lâmpada se uma responder
		if sala.TemLampada {
			statusLampada := "🌑 APAGADA"
			if sala.LampadaLigada {
				statusLampada = "💡 ACESA"
			}
			blocos = append(blocos, fmt.Sprintf("💡 Lâmpada: %s", statusLampada))
		}

		// Só mostra Eventos se for uma sala de Catraca/NFC
		if sala.TemCatraca {
			blocos = append(blocos, fmt.Sprintf("🪪 Acesso: %s", sala.UltimoEvento))
		}

		// Caso os dados ainda estejam a chegar e não haja flags
		if len(blocos) == 0 {
			blocos = append(blocos, "⏳ A aguardar identificação dos dispositivos...")
		}

		// Imprime o ID da sala numa linha e os dados identados abaixo
		fmt.Printf("📍 [%s]\n   ↳ %s\n\n", id, strings.Join(blocos, " | "))
	}
	fmt.Println("===============================")
}
