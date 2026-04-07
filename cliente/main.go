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

// Estado consolidado de cada sala mantido pelo dashboard.
type EstadoSala struct {
	TemSensorTemp     bool
	TemperaturaAtual  float64
	TemperaturaAlvo   float64
	UltimaLeituraTemp time.Time // Ultima telemetria recebida do sensor

	TemAC    bool
	ArLigado bool
	ModoAuto bool

	TemLampada    bool
	LampadaLigada bool

	TemCatraca           bool
	UltimoEvento         string
	UltimaLeituraCatraca time.Time // Ultimo evento recebido da catraca
}

// Estados das salas protegidos por mutex.
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
		addrEnv = "localhost:8083"
	}

	conn, err := net.Dial("tcp", addrEnv)
	if err != nil {
		fmt.Printf("❌ Erro ao ligar ao Integrador em %s: %v\n", addrEnv, err)
		return
	}
	defer conn.Close()

	fmt.Println("✅ Ligado com sucesso ao Gateway Integrador!")

	// Leitor de rede e motor de regras rodam em segundo plano.
	go ouvirRedeEProcessarLogica(conn)

	// Remove salas sem dispositivos ativos para manter o painel enxuto.
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

			// Sincroniza o modo manual com os outros clientes.
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
				// Notifica os demais clientes sobre a ativacao do modo automatico.
				fmt.Fprintf(conn, "SYNC|%s|AUTO\n", idSala)
				fmt.Println("✅ Modo Automático ATIVADO para a sala", idSala)
			} else {
				// Notifica os demais clientes sobre a desativacao do modo automatico.
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

// Processa mensagens vindas do integrador e atualiza o estado local.
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

		// Telemetria UDP de temperatura.
		if tipoMsg == "TLM" && partes[1] == "T" {
			idSala := partes[2]
			tempAtual, _ := strconv.ParseFloat(partes[3], 64)

			sala := getSalaSegura(idSala)
			sala.TemSensorTemp = true
			sala.TemperaturaAtual = tempAtual
			sala.UltimaLeituraTemp = time.Now()

			avaliarModoAutomatico(idSala, sala, conn)
		}

		// Evento de acesso vindo da catraca.
		if tipoMsg == "EVT" {
			idSala := partes[2]
			evento := partes[3]

			sala := getSalaSegura(idSala)
			sala.TemCatraca = true
			sala.UltimoEvento = evento
			sala.UltimaLeituraCatraca = time.Now()
		}

		// Confirmacao de atuador recebida do integrador.
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

		// Sincronizacao vinda de outro cliente para manter o estado consistente.
		if tipoMsg == "SYNC" && len(partes) >= 3 {
			idSala := partes[1]
			modo := partes[2]

			sala := getSalaSegura(idSala)
			sala.ModoAuto = (modo == "AUTO")
		}

		// Erro de equipamento recebido do gateway.
		if tipoMsg == "ERRO" && len(partes) >= 3 {
			origem := partes[1]
			detalhe := partes[2]

			fmt.Printf("\n❌ [FALHA DE COMANDO - %s] %s\n", origem, detalhe)

			if strings.HasPrefix(detalhe, "Atuador ") {
				pedacos := strings.Split(detalhe, " ")
				if len(pedacos) >= 2 {
					idAtuador := pedacos[1]
					idSala := strings.TrimPrefix(idAtuador, "AC_")
					idSala = strings.TrimPrefix(idSala, "LED_")

					sala := getSalaSegura(idSala)

					// Marca o atuador como indisponivel no estado local.
					if strings.HasPrefix(idAtuador, "AC_") {
						sala.TemAC = false
					} else if strings.HasPrefix(idAtuador, "LED_") {
						sala.TemLampada = false
					}

					// Se o modo automatico estava ativo, desativa por seguranca.
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

// Periodicamente remove salas sem dispositivos ativos.
func limparDispositivosInativos() {
	for {
		time.Sleep(3 * time.Second)

		mu.Lock()
		for id, sala := range salas {
			// Sensor de temperatura sem sinal recente deixa de ser exibido como ativo.
			if sala.TemSensorTemp && time.Since(sala.UltimaLeituraTemp) > 5*time.Second {
				sala.TemSensorTemp = false
			}

			// A catraca possui janela maior de tolerancia por ser mais lenta.
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

// Aplica a regra de controle automatico do ar condicionado.
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

// Imprime o estado atual das salas e apenas os dispositivos detectados.
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

		// Se ainda nao houver blocos, a sala ainda aguarda deteccao ou confirmacao.
		if len(blocos) == 0 {
			blocos = append(blocos, "⏳ A aguardar identificação dos dispositivos...")
		}

		fmt.Printf("📍 [%s]\n   ↳ %s\n\n", id, strings.Join(blocos, " | "))
	}
	fmt.Println("===============================")
}
