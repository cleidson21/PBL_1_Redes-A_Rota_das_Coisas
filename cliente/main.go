package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ESTRUTURA DE DADOS (O "Cérebro" do Sistema)
type EstadoSala struct {
	// Flags e Timestamps para saber quando os dispositivos "morreram"
	TemSensorTemp     bool
	TemperaturaAtual  float64
	TemperaturaAlvo   float64
	UltimaLeituraTemp time.Time // Guarda o último sinal de vida do termômetro

	TemAC    bool
	ArLigado bool
	ModoAuto bool

	TemLampada    bool
	LampadaLigada bool

	TemCatraca           bool
	UltimoEvento         string
	UltimaLeituraCatraca time.Time // Guarda o último sinal de vida da catraca
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
			ModoAuto:         false,
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

	// 1. INICIA O CÉREBRO EM SEGUNDO PLANO
	go ouvirRedeEProcessarLogica(conn)

	// 2. INICIA O "FAXINEIRO" DE DISPOSITIVOS FANTASMAS (Garbage Collector)
	go limparDispositivosInativos()

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

			// SINCRONIZA O RESTO DA REDE QUE ENTROU EM MODO MANUAL
			fmt.Fprintf(conn, "SYNC|%s|MANUAL\n", idSala)
			fmt.Fprintf(conn, "AC_%s|%s\n", idSala, acao)
			fmt.Println("⏳ Comando enviado! (Sincronizando modo MANUAL com a rede...)")

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
				// AVISA TODOS OS OUTROS CLIENTES
				fmt.Fprintf(conn, "SYNC|%s|AUTO\n", idSala)
				fmt.Println("✅ Modo Automático ATIVADO para a sala", idSala)
			} else {
				// AVISA TODOS OS OUTROS CLIENTES
				fmt.Fprintf(conn, "SYNC|%s|MANUAL\n", idSala)
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

		// 1. RECEBEU TELEMETRIA (Sensor UDP)
		if tipoMsg == "TLM" && partes[1] == "T" {
			idSala := partes[2]
			tempAtual, _ := strconv.ParseFloat(partes[3], 64)

			sala := getSalaSegura(idSala)
			sala.TemSensorTemp = true
			sala.TemperaturaAtual = tempAtual
			sala.UltimaLeituraTemp = time.Now() // Atualiza o sinal de vida!

			avaliarModoAutomatico(idSala, sala, conn)
		}

		// 2. RECEBEU EVENTO (Sensor TCP / Catraca)
		if tipoMsg == "EVT" {
			idSala := partes[2]
			evento := partes[3]

			sala := getSalaSegura(idSala)
			sala.TemCatraca = true
			sala.UltimoEvento = evento
			sala.UltimaLeituraCatraca = time.Now() // Atualiza o sinal de vida!
		}

		// 3. RECEBEU CONFIRMAÇÃO DO ATUADOR (ACK)
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

		// 4. RECEBEU SINCRONIZAÇÃO DE OUTRO CLIENTE (Evita o Split-Brain)
		if tipoMsg == "SYNC" && len(partes) >= 3 {
			idSala := partes[1]
			modo := partes[2] // "AUTO" ou "MANUAL"

			sala := getSalaSegura(idSala)
			sala.ModoAuto = (modo == "AUTO")
		}

		// 5. RECEBEU ERRO DE EQUIPAMENTO (Disjuntor e Limpeza de Fantasmas)
		if tipoMsg == "ERRO" && len(partes) >= 3 {
			origem := partes[1]  // De onde veio o erro (GATEWAY, AC ou LED)
			detalhe := partes[2] // O texto do erro

			fmt.Printf("\n❌ [FALHA DE COMANDO - %s] %s\n", origem, detalhe)

			if strings.HasPrefix(detalhe, "Atuador ") {
				pedacos := strings.Split(detalhe, " ")
				if len(pedacos) >= 2 {
					idAtuador := pedacos[1] // Ex: AC_SALA_1
					idSala := strings.TrimPrefix(idAtuador, "AC_")
					idSala = strings.TrimPrefix(idSala, "LED_")

					sala := getSalaSegura(idSala)

					// MARCA O ATUADOR COMO OFFLINE/INEXISTENTE
					if strings.HasPrefix(idAtuador, "AC_") {
						sala.TemAC = false
					} else if strings.HasPrefix(idAtuador, "LED_") {
						sala.TemLampada = false
					}

					// SE O ERRO CAUSOU DESARME, AVISA A REDE INTEIRA!
					if sala.ModoAuto {
						sala.ModoAuto = false
						fmt.Fprintf(conn, "SYNC|%s|MANUAL\n", idSala)
						fmt.Printf("🛑 MODO AUTOMÁTICO da [%s] foi DESATIVADO na rede por segurança.\n", idSala)
					}
				}
			}

			fmt.Print("Escolha uma opção: ")
		}

		mu.Unlock()
	}
}

// O FAXINEIRO (Garbage Collector de Dispositivos Offline)
func limparDispositivosInativos() {
	for {
		time.Sleep(3 * time.Second) // Varre a memória a cada 3 segundos

		mu.Lock()
		for id, sala := range salas {
			// Se o sensor de temperatura não manda nada há mais de 5 segundos, considera offline
			if sala.TemSensorTemp && time.Since(sala.UltimaLeituraTemp) > 5*time.Second {
				sala.TemSensorTemp = false
			}

			// A catraca é mais lenta, damos 30 segundos de tolerância antes de a esconder
			if sala.TemCatraca && time.Since(sala.UltimaLeituraCatraca) > 30*time.Second {
				sala.TemCatraca = false
			}

			if !sala.TemSensorTemp && !sala.TemCatraca && !sala.TemAC && !sala.TemLampada {
				delete(salas, id)
			}
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
		fmt.Println("Nenhum dado recebido ainda. Aguarde pelos sensores ou envie comandos...")
		return
	}

	for id, sala := range salas {
		var blocos []string

		if sala.TemSensorTemp {
			blocos = append(blocos, fmt.Sprintf("🌡️ Temp: %.1f°C", sala.TemperaturaAtual))
			blocos = append(blocos, fmt.Sprintf("🎯 Alvo: %.1f°C", sala.TemperaturaAlvo))

			modo := "✋ MANUAL"
			if sala.ModoAuto {
				modo = "🤖 AUTO"
			}
			blocos = append(blocos, fmt.Sprintf("⚙️ Modo: %s", modo))
		}

		if sala.TemAC {
			statusAr := "🔴 DESLIGADO"
			if sala.ArLigado {
				statusAr = "🟢 LIGADO"
			}
			blocos = append(blocos, fmt.Sprintf("❄️ Ar: %s", statusAr))
		}

		if sala.TemLampada {
			statusLampada := "🌑 APAGADA"
			if sala.LampadaLigada {
				statusLampada = "💡 ACESA"
			}
			blocos = append(blocos, fmt.Sprintf("💡 Lâmpada: %s", statusLampada))
		}

		if sala.TemCatraca {
			blocos = append(blocos, fmt.Sprintf("🪪 Acesso: %s", sala.UltimoEvento))
		}

		// Graças ao Faxineiro (limparDispositivosInativos), se isto aparecer é porque
		// a sala está a ser construída ou aguarda um ACK. E se não vier nada, ela some!
		if len(blocos) == 0 {
			blocos = append(blocos, "⏳ A aguardar identificação dos dispositivos...")
		}

		fmt.Printf("📍 [%s]\n   ↳ %s\n\n", id, strings.Join(blocos, " | "))
	}
	fmt.Println("===============================")
}
