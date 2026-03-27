#!/bin/bash

# Defina o IP da Máquina 1 (Gateway/Integrador)
IP_GATEWAY="172.16.103.8" 
QTD_SALAS=50

echo "🧠 Iniciando CÉREBROS (Clientes) de controle para $QTD_SALAS salas..."
echo "Alvo: $IP_GATEWAY"

for i in $(seq 1 $QTD_SALAS); do
    # 5. Cliente (usando -td para criar um pseudo-terminal e não bugar o menu os.Stdin)
    docker run -td --name "stress_cliente_$i" \
        -e INTEGRADOR_ADDR="$IP_GATEWAY:8083" \
        cleidsonramos/cliente:v1 > /dev/null
done

echo "✅ $QTD_SALAS Clientes ouvindo a rede e processando histerese!"
echo "💡 Dica: Para ver um painel funcionando, digite: docker attach stress_cliente_1"