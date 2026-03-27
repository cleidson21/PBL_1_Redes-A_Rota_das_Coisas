#!/bin/bash

echo "🚀 Iniciando o Integrador Gateway..."

# Limpa o container antigo silenciosamente para evitar conflito de nomes
docker rm -f integrador_pbl 2>/dev/null

# Sobe o Integrador mapeando todas as portas necessárias da arquitetura
docker run -d --name integrador_pbl \
    -p 8080:8080/udp \
    -p 8081:8081/tcp \
    -p 8082:8082/tcp \
    -p 8083:8083/tcp \
    cleidsonramos/integrador:v1

echo "✅ Integrador iniciado com sucesso em background!"
echo "💡 Dica: Para acompanhar os logs em tempo real, digite: docker logs -f integrador_pbl"