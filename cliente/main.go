package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	// Busca o IP do integrador (no Docker será "integrador:8082")
	addrEnv := os.Getenv("INTEGRADOR_ADDR")
	if addrEnv == "" {
		addrEnv = "localhost:8082"
	}

	// Inicia a conexão TCP com o Integrador
	conn, err := net.Dial("tcp", addrEnv)
	if err != nil {
		fmt.Printf("❌ Erro ao conectar ao Integrador em %s: %v\n", addrEnv, err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)         // Lê o que você digita no teclado
	serverReader := bufio.NewReader(conn)       // Lê o que o Integrador responde

	fmt.Println("✅ Conectado com sucesso ao Sistema Integrador!")

	for {
		fmt.Println("\n===================================")
		fmt.Println("❄️  PAINEL DE CONTROLE - IoT ❄️")
		fmt.Println("===================================")
		fmt.Println("[1] Obter Status Atual")
		fmt.Println("[2] Ativar Modo Automático")
		fmt.Println("[3] Ativar Modo Manual")
		fmt.Println("[4] Ligar Ar-Condicionado (Manual)")
		fmt.Println("[5] Desligar Ar-Condicionado (Manual)")
		fmt.Println("[6] Definir Temperatura Alvo")
		fmt.Println("[0] Sair do Painel")
		fmt.Println("===================================")
		fmt.Print("Escolha uma opção: ")

		// Lê a opção digitada
		opcao, _ := reader.ReadString('\n')
		opcao = strings.TrimSpace(opcao)

		var comando string

		// Traduz a opção do menu para o comando de rede que criamos no Integrador
		switch opcao {
		case "1":
			comando = "STATUS"
		case "2":
			comando = "AUTO"
		case "3":
			comando = "MANUAL"
		case "4":
			comando = "LIGAR"
		case "5":
			comando = "DESLIGAR"
		case "6":
			fmt.Print("Digite a nova temperatura alvo (ex: 22.5): ")
			temp, _ := reader.ReadString('\n')
			comando = "SET_ALVO " + strings.TrimSpace(temp)
		case "0":
			fmt.Println("Desconectando do sistema...")
			return
		default:
			fmt.Println("⚠️ Opção inválida, tente novamente.")
			continue
		}

		// Envia o comando escolhido pela rede (TCP)
		fmt.Fprintf(conn, "%s\n", comando)

		// Aguarda e imprime a resposta oficial do Integrador
		resposta, _ := serverReader.ReadString('\n')
		fmt.Printf("\n📩 Resposta do Sistema: %s", resposta)
	}
}